// the "modify" command modifies an exisitng Go installation to support x-trace
// by adding goroutine-local variables and a way to access them.
// run "go run modify.go" and then "go install -a std"
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"runtime"
)

var localFieldName = "local"

var local = "local"
var newproc1 = "newproc1"
var newproc2 = "newproc2"

// Walks the AST looking for the declaration of "newproc" then calls modifyNewProc
func findAndModifyNewProcDeclaration(f *ast.File) {
	// Find declaration of 'newproc'
	var newproc *ast.FuncDecl
	for _, decl := range(f.Decls) {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "newproc" && funcDecl.Recv == nil {
			newproc = funcDecl
		}
	}

	// Could not find declaration of newproc
	if newproc == nil { return }

	// The temporary variable we use in the code
	tempVar := ast.NewIdent("newlocal")

	// Create the new assign statement
	//   newlocal := derivelocal(getg().local)
	assignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{tempVar},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: ast.NewIdent("derivelocal"),
				Args: []ast.Expr{&ast.SelectorExpr{
					X: &ast.CallExpr{Fun: ast.NewIdent("getg")},
					Sel: ast.NewIdent(localFieldName),
				}},
			},
		},
	}

	// Find the invocation of systemstack.
	// Insert the assignStmt before it
	// Modify the call to newproc1 within systemstack

	for i, stmt := range(newproc.Body.List) {
		exprStmt, ok2 := stmt.(*ast.ExprStmt)
		if !ok2 { continue }

		callExpr, ok3 := exprStmt.X.(*ast.CallExpr)
		if !ok3 { continue }

		ident, ok4 := callExpr.Fun.(*ast.Ident)
		if !ok4 || ident.Name != "systemstack" { continue }

		// Add the additional assignment statement
		var newStmts []ast.Stmt
		newStmts = append(newStmts, newproc.Body.List[:i]...)
		newStmts = append(newStmts, assignStmt)
		newStmts = append(newStmts, newproc.Body.List[i:]...)
		newproc.Body.List = newStmts

		// Within systemstack, rewrite the call of newproc1 to instead call newproc2 and pass local
		ast.Inspect(callExpr, func (n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok { return true }

			ident, ok2 := callExpr.Fun.(*ast.Ident)
			if !ok2 || ident.Name != newproc1 { return true }

			callExpr.Fun = ast.NewIdent(newproc2)
			callExpr.Args = append(callExpr.Args, tempVar)
			return false
		})

		break
	}

}

// Walks the AST looking for the declaration of "newproc1" then modifies it in the following ways:
//  1) create a new function called "newproc2" with the body of newproc1 and additional argument "local"
//  2) newproc1 proxies to newproc2, passing 'nil' for the 'local' argument
//  3) additional statement within the body of "newproc2" that sets newg.local to the passed 'local' variable
func findAndModifyNewProc1Declaration(f *ast.File) {
	var newproc1 *ast.FuncDecl
	var i int

	for ; i < len(f.Decls); i++ {
		funcDecl, ok := f.Decls[i].(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "newproc1" {
			newproc1 = funcDecl
			break
		}
	}

	// Could not find a declaration of newproc1
	if newproc1 == nil { return }

	// Create the new declarations
	newNewproc1 := makeNewNewproc1(newproc1)
	newproc2 := makeNewproc2(newproc1)

	// Insert into the src, removing the old newproc1
	var decls []ast.Decl
	decls = append(decls, f.Decls[:i]...)
	decls = append(decls, newNewproc1)
	decls = append(decls, newproc2)
	decls = append(decls, f.Decls[i+1:]...)
	f.Decls = decls
}

// Creates a new declaration of newproc1 that looks as follows:
//		func newproc1(<original params>) <original return type> {
//			return newproc2(<original params>, nil)
//		}
func makeNewNewproc1(newproc1 *ast.FuncDecl) *ast.FuncDecl {
	// All parameters of newproc1 are arguments to newproc2, plus 'nil' for the extra 'local' argument
	var args []ast.Expr
	for _, arg := range(newproc1.Type.Params.List) {
		for _, name := range(arg.Names) {
			args = append(args, ast.NewIdent(name.Name))
		}
	}
	args = append(args, ast.NewIdent("nil"))

	// Body of our new function simply invokes newproc2
	invocationStmt := &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.CallExpr{
				Fun: ast.NewIdent(newproc2),
				Args: args ,
			},
		},
	}

	// Construct and return new function declaration
	return &ast.FuncDecl{
		Doc: newproc1.Doc,
		Recv: newproc1.Recv,
		Name: newproc1.Name,
		Type: newproc1.Type,
		Body: &ast.BlockStmt{List: []ast.Stmt{invocationStmt}},
	}
}

// Creates the function newproc2, based off existing newproc1
//		func newproc2(fn *funcval, argp *uint8, narg int32, nret int32, callerpc uintptr, local interface{}) *g {
//			...
//		}
// newproc2 has the same method body as newproc1, with an additional line midway with:
// 		newg.local = local
func makeNewproc2(newproc1 *ast.FuncDecl) *ast.FuncDecl {
	// Arguments of newproc2 are same as newproc1 plus an extra:
	//    func newproc2(<newproc1args>, local interface{})
	var args []*ast.Field
	args = append(args, newproc1.Type.Params.List...)
	args = append(args, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent(local)},
		Type: &ast.InterfaceType{Methods: &ast.FieldList{}},
	})

	// The extra statement we want to insert into the body of newproc2, that copies the local field
	assignStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{
			&ast.SelectorExpr{
				X: ast.NewIdent("newg"),
				Sel: ast.NewIdent(localFieldName),
			},
		},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{ast.NewIdent(local)},
	}

	// Copy newproc1's body and insert the assign statement prior to the invocation of gostartcallfn:
	//   newg.local = local
	var stmts []ast.Stmt
	for _, stmt := range(newproc1.Body.List) {
		exprStmt, ok := stmt.(*ast.ExprStmt)
		if ok {
			callExpr, ok2 := exprStmt.X.(*ast.CallExpr)
			if ok2 {
				ident, ok3 := callExpr.Fun.(*ast.Ident)
				if ok3 && ident.Name == "gostartcallfn" {
					stmts = append(stmts, assignStmt)
				}
			}
		}
		stmts = append(stmts, stmt)
	}

	// Return the new function declaration
	return &ast.FuncDecl{
		Name: ast.NewIdent(newproc2),
		Doc: newproc1.Doc,
		Recv: newproc1.Recv,
		Body: &ast.BlockStmt{List: stmts},
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: args},
			Results: newproc1.Type.Results,
		},
	}

	//// Insert the new function before the current newproc1 decl
	//var newDecls []ast.Decl
	//for _, decl := range(f.Decls) {
	//	newDecls = append(newDecls, decl)
	//	if decl == newproc1 {
	//		newDecls = append(newDecls, newproc2)
	//	}
	//}
	//f.Decls = newDecls
}

// Walks the AST looking for a function called "newproc2", which only exists if we already modifed the file
func procDotGoAlreadyModified(f *ast.File) bool {
	newproc2Exists := false

	// Tests an AST node to see if it's a function declaration called "newproc2"
	doesNewProc2Exist := func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if ok && funcDecl.Name.Name == "newproc2" {
			newproc2Exists = true
			return false
		}
		return true
	}

	ast.Inspect(f, doesNewProc2Exist)
	return newproc2Exists
}

func modifyProcDotGo() {
	goroot := runtime.GOROOT()
	path := goroot + "/src/runtime/proc.go"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse proc.go in ", path, ": ", err)
		os.Exit(1)
	}

	if procDotGoAlreadyModified(f) {
		fmt.Println("proc.go has already been modified; skipping")
		return
	}

	// Modify the declaration of newproc
	findAndModifyNewProcDeclaration(f)
	findAndModifyNewProc1Declaration(f)

	//printer.Fprint(os.Stdout, token.NewFileSet(), f)
	//fmt.Println()

	outfile, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open", path, err)
		os.Exit(1)
	}

	err = format.Node(outfile, fset, f)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write to", path, err)
		os.Exit(1)
	}
	outfile.Close()
}

func modifyRuntime2dotGo() {
	goroot := runtime.GOROOT()
	path := goroot + "/src/runtime/runtime2.go"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse runtime2.go in ", path, ": ", err)
		os.Exit(1)
	}

	alreadyModified := false
	continueStepping := true

	ast.Inspect(f, func(n ast.Node) bool {
		typeDecl, ok := n.(*ast.TypeSpec)
		if ok {
			if typeDecl.Name.Name == "g" {
				fmt.Print("Found g struct...")
				gStruct := typeDecl.Type.(*ast.StructType)

				for _, field := range gStruct.Fields.List {
					for _, name := range field.Names {
						if name.Name == localFieldName {
							//local already exists
							fmt.Println("...already modified.")
							continueStepping = false
							alreadyModified = true
						}
					}
				}
				if !alreadyModified {
					docComment := &ast.Comment{Text: "\n// Goroutine-local storage"}

					gStruct.Fields.List = append(gStruct.Fields.List, &ast.Field{
						Names: []*ast.Ident{ast.NewIdent(localFieldName)},
						Type: &ast.InterfaceType{
							Methods: &ast.FieldList{},
						},
						Doc: &ast.CommentGroup{
							[]*ast.Comment{
								docComment,
							},
						},
					})

					fmt.Println("...Created modified goroutine structure:")
					printer.Fprint(os.Stdout, fset, gStruct)
					fmt.Println()
					continueStepping = false
				}
			} else {
				return false
			}
		}
		return continueStepping
	})

	if !alreadyModified {
		outfile, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open", path, err)
			os.Exit(1)
		}

		err = format.Node(outfile, fset, f)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to write to", path, err)
			os.Exit(1)
		}
		outfile.Close()
	}
}

var localMethodsText = `package runtime

type Local interface {
	Derive() Local
}

func GetLocal() Local {
    l, ok := getg().local.(Local)
    if ok { return l }
    return nil
}

func SetLocal(local Local) {
    getg().local = local
}

func GetGoID() int64 {
	return getg().goid
}

func derivelocal(oldv interface{}) interface{} {
	local, ok := oldv.(Local)
	if !ok { return nil }
	return local.Derive()
}
`

func addMethodsToRuntime() {
	fmt.Print("Creating local access methods...")
	goroot := runtime.GOROOT()
	path := goroot + "/src/runtime/" + localFieldName + ".go"
	newfile, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create", path, err)
		os.Exit(1)
	}
	_, err = newfile.WriteString(localMethodsText)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write to", path, err)
		os.Exit(1)
	}
	err = newfile.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save ", path, err)
		os.Exit(1)
	}
	fmt.Println("...Done")
}

func main() {
	modifyProcDotGo()
	modifyRuntime2dotGo()
	addMethodsToRuntime()
}
