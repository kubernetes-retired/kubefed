/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
)

var (
	in      = flag.String("in", "", "input")
	out     = flag.String("out", "", "output")
	pkgName = flag.String("pkg", "", "package")
)

func main() {
	flag.Parse()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *in, nil, 0)
	if err != nil {
		log.Fatal(err)
	}

	conf := types.Config{Importer: importer.Default()}

	pkg, err := conf.Check(*pkgName, fset, []*ast.File{f}, nil)
	if err != nil {
		log.Fatal(err)
	}
	if err := ioutil.WriteFile(*out, []byte(fmt.Sprintf("package %s\nconst OK = true", pkg.Name())), 0666); err != nil {
		log.Fatal(err)
	}
}
