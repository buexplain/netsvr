/**
* Copyright 2022 buexplain@qq.com
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package wd

import (
	"github.com/lesismal/nbio/logging"
	"os"
	"path/filepath"
	"strings"
)

// RootPath 程序根目录
var RootPath string

func init() {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("Get process working directory failed：%s", err)
		os.Exit(1)
	}
	RootPath = strings.TrimSuffix(filepath.ToSlash(dir), "/") + "/"
}
