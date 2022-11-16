package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	PACKAGE        = "go"
	E_SYNTAX_ERROR = "file [%s]\n\tline [%d]: syntax error"
)

func GeneratePackage(arpc_meta *ArpcMeta, filename string) string {
	path := "./"
	if len(arpc_meta.Package) > 0 {
		for _, pkg := range arpc_meta.Package {
			if pkg.Language == PACKAGE {
				path = pkg.Path
			}
		}
	}
	// mkdir
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
	_, fileName := filepath.Split(filename)
	go_file := strings.Split(fileName, ".")[0] + "_arpc.go"

	localtime := time.Now().Format("1949-10-01 15:00:00")

	file_str := fmt.Sprintf("//%s\n\n", localtime)

	file_str += fmt.Sprintf("package %s\n\n", path)

	for k, v := range arpc_meta.Param {
		result := GenerateParamStruct(k, v)
		file_str += result + "\n\n"
	}

	if _, err := os.Stat(go_file); err == nil {
		os.Remove(go_file)
	}

	f, err := os.Create(go_file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = io.WriteString(f, file_str)
	if err != nil {
		log.Fatal(err)
	}
	return go_file
}

// {'version': '1.0', 'package': [{'language': 'python', 'path': 'api'}], 'procedures': [{'name': 'GetUserV1', 'request': 'RequestV1', 'response': 'ResponseV1'}], 'param': {'RequestV1': [{'name': 'user_id', 'type': 'integer', 'index': 1}], 'ResponseV1': [{'name': 'user_id', 'type': 'integer', 'index': 1}, {'name': 'username', 'type': 'string', 'index': 2}]}}

func CompileArpc(path string) (*ArpcMeta, error) {
	var err error

	var arpc_meta ArpcMeta = ArpcMeta{}

	// 当前解析到的行号
	var line_num int = 0
	// 正在解析 package
	var handle_package bool = false
	// 正在解析 procedure
	var handle_procedures bool = false
	// 正在解析 param
	var handle_param bool = false
	// 当前解析 param 名
	var param_name string = ""

	var file *os.File
	file, err = os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		fmt.Println("打开文件出错：", err)
	}
	defer file.Close()

	var content []byte
	var buf []byte = make([]byte, 128)

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			fmt.Println("文件读取完毕")
			break
		}
		if err != nil {
			log.Fatal(err)
			break
		}
		content = append(content, buf[:n]...)
	}
	var lines = strings.Split(string(content), "\n")

	for _, line := range lines {
		line_num++
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if handle_package {
			if strings.HasPrefix(line, "}") {
				handle_package = false
				continue
			} else if strings.HasPrefix(line, "{") {
				continue
			} else {
				// 正则匹配 language: path
				var reg = regexp.MustCompile(`^(.*):\s*(.*)`)
				var match = reg.FindStringSubmatch(line)
				if len(match) == 3 {
					arpc_meta.Package = append(arpc_meta.Package, Package{
						Language: match[1],
						Path:     match[2],
					})
				} else {
					return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
				}
			}
		} else if handle_procedures {
			if strings.HasPrefix(line, "}") {
				handle_procedures = false
				continue
			} else if strings.HasPrefix(line, "{") {
				continue
			} else {
				// 正则匹配 name: request response
				var reg = regexp.MustCompile(`^procedure\s+(\w+)\s*\((\w+)\):\s*(\w+)`)
				var match = reg.FindStringSubmatch(line)
				for _, v := range arpc_meta.Procedures {
					if v.Name == match[1] {
						return nil, fmt.Errorf("file [%s]\n\tline [%d]: repeated procedure: %s", path, line_num, match[1])
					}
				}
				if len(match) == 4 {
					arpc_meta.Procedures = append(arpc_meta.Procedures, Procedures{
						Name:     match[1],
						Request:  match[2],
						Response: match[3],
					})
				} else {
					return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
				}
			}
		} else if handle_param {
			if strings.HasPrefix(line, "}") {
				handle_param = false
				continue
			} else if strings.HasPrefix(line, "{") {
				continue
			} else {
				// 正则匹配 name: type = index
				var reg = regexp.MustCompile(`^\s*(\w+):\s*(\w+)\s*=\s*(\d+)`)
				var match = reg.FindStringSubmatch(line)
				if len(match) == 4 {
					index, err := strconv.Atoi(match[3])
					if err != nil {
						return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
					}
					arpc_meta.Param[param_name] = append(arpc_meta.Param[param_name], Param{
						Name:  match[1],
						Type:  match[2],
						Index: index,
					})
				} else {
					return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
				}
			}
		} else {
			if strings.HasPrefix(line, "package") {
				if arpc_meta.Package != nil {
					return nil, fmt.Errorf("file [%s]\n\tline [%d]: repeated package area", path, line_num)
				}
				handle_package = true
				arpc_meta.Package = make([]Package, 0)
				continue
			} else if strings.HasPrefix(line, "procedures") {
				if arpc_meta.Procedures != nil {
					return nil, fmt.Errorf("file [%s]\n\tline [%d]: repeated procedures area", path, line_num)
				}
				handle_procedures = true
				arpc_meta.Procedures = make([]Procedures, 0)
				continue
			} else if strings.HasPrefix(line, "param") {
				// 正则匹配 param RequestV1 {
				var reg = regexp.MustCompile(`^param\s+(\w+)\s+{`)
				var match = reg.FindStringSubmatch(line)
				if len(match) == 2 {
					param_name = match[1]
					if arpc_meta.Param[param_name] != nil {
						return nil, fmt.Errorf("file [%s]\n\tline [%d]: repeated param", path, line_num)
					}
					handle_param = true
					// arpc_meta.Param is nil?
					if arpc_meta.Param == nil {
						arpc_meta.Param = make(map[string][]Param)
					}
					arpc_meta.Param[param_name] = make([]Param, 0)
				} else {
					return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
				}
			} else if strings.HasPrefix(line, "arpc") {
				// 正则匹配 arpc: *
				var reg = regexp.MustCompile(`^arpc:\s*(.*)`)
				var match = reg.FindStringSubmatch(line)
				if len(match) == 2 {
					arpc_meta.Version = match[1]
				} else {
					return nil, fmt.Errorf(E_SYNTAX_ERROR, path, line_num)
				}
			}
		}
	}

	return &arpc_meta, nil
}

func Compile(path string) {
	fmt.Println("========================================")
	var res, err = CompileArpc(path)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v \n", res)
	GeneratePackage(res, path)
	fmt.Println("========================================")
	// for k, v := range res.Param {
	// 	result := GenerateParamStruct(k, v)
	// 	fmt.Println(result)
	// }
}

func Compiles(paths []string) {
	for _, path := range paths {
		fmt.Println("========================================")
		var res, err = CompileArpc(path)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%+v \n", res)
		GeneratePackage(res, path)
		fmt.Println("========================================")
		// for k, v := range res.Param {
		// 	result := GenerateParamStruct(k, v)
		// 	fmt.Println(result)
		// }
	}
}
