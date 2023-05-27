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

package apis

import (
	"canary-crd/pkg/apis/app/v1"
)

//这段代码是在 Go 语言中的 init 函数，它在每个 Go 程序中都会在 main 函数之前被自动执行。init 函数主要用于执行初始化工作，例如初始化变量，注册函数等。
//
//在这段代码中，init 函数的作用是将 v1.SchemeBuilder.AddToScheme 函数添加到 AddToSchemes 列表中。
//这是 Kubernetes 中的一种机制，用于将特定的 API 组（在这个例子中是 app/v1）的资源类型注册到 Scheme 中。
//
//Scheme 是 Kubernetes 中用于识别和映射 API 对象的一种机制。当 Kubernetes 需要处理 API 对象（例如 Pod，Service 等）时，
//它需要知道这些对象的完全限定名（包括 API 组，版本和种类）。Scheme 就是用来保存这些信息的。
//
//v1.SchemeBuilder.AddToScheme 是一个函数，它的作用是将 app/v1 API 组的所有资源类型添加到一个 Scheme 中。
//通过将这个函数添加到 AddToSchemes 列表中，我们就可以在需要的时候将 app/v1 API 组的所有资源类型添加到任何 Scheme 中。
//
//总的来说，这段代码的作用是注册 app/v1 API 组的资源类型，以便在处理这些资源时可以正确地识别和映射它们。

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
