/*
Copyright 2019 Hypo.

Licensed under the GNU General Public License, Version 3 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/Coderhypo/canary-crd/blob/master/LICENSE

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Generate deepcopy for apis
//go:generate go run ../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./... -h ../../hack/boilerplate.go.txt

// Package apis contains Kubernetes API groups.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
)

//这个文件定义了AddToSchemes变量和AddToScheme函数。AddToSchemes是一个runtime.SchemeBuilder类型的变量，
//它可以用来将定义在项目中的所有资源添加到一个 Scheme 中。AddToScheme函数则是一个将所有资源添加到给定 Scheme 的函数，
//它通过调用AddToSchemes.AddToScheme方法来实现这个功能。

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
