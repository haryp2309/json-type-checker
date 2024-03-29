package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang-collections/collections/set"
	"github.com/valyala/fastjson"
)

func FindFilesByRegex(directoryPath string, regex *regexp.Regexp, onFileFound func(filePath string)) error {
	err := filepath.WalkDir(directoryPath, func(path string, d fs.DirEntry, err error) error {
		validFilename := regex.MatchString(d.Name())
		isFile := !d.IsDir()
		if err != nil {
			return err
		}
		if validFilename && isFile {
			onFileFound(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil

}

func FindTypedefFiles(directoryPath string, onFileFound func(filePath string)) error {
	libRegEx, err := regexp.Compile(`^.+\.typedef\.json$`)
	if err != nil {
		return err
	}
	err = FindFilesByRegex(directoryPath, libRegEx, onFileFound)
	if err != nil {
		return err
	}
	return nil
}

func FindJsonFiles(typedefPath string, onFileFound func(filePath string)) error {
	filename, was_replaced := strings.CutSuffix(typedefPath, ".typedef.json")
	if !was_replaced {
		return errors.New("suffix wasn't removed properly")
	}

	jsonFilename := filename + ".json"
	if _, err := os.Stat(jsonFilename); err == nil {
		onFileFound(jsonFilename)
	}
	return nil
}

type Node struct {
	Type       string           `json:"type"`
	Children   *Node            `json:"children,omitempty"`
	Properties map[string]*Node `json:"properties,omitempty"`
	Define     map[string]*Node `json:"define,omitempty"`
	Optional   bool             `json:"optional,omitempty"`
}

func ParseTypedefToNode(typedefFile []byte) (Node, error) {
	var obj Node
	err := json.Unmarshal(typedefFile, &obj)
	return obj, err
}

func PrintMessage(message string) {
	fmt.Println("[JTC]: " + message)
}

func MergeDefinitions(definition1 map[string]*Node, definition2 map[string]*Node) map[string]*Node {
	if definition2 == nil {
		return definition1
	}
	result := make(map[string]*Node)
	for _, definition := range [2]map[string]*Node{definition1, definition2} {
		for key, value := range definition {
			result[key] = value
		}
	}
	return result
}

func ValidateJsonFile(valNode Node, jsonObj *fastjson.Value, jsonPath string, definition map[string]*Node) {

	mergedDefinition := MergeDefinitions(definition, valNode.Define)

	switch valNode.Type {
	case "string":
		_, err := jsonObj.StringBytes()
		if err != nil {
			PrintMessage("❌ Expected string at " + jsonPath)
			return
		}
	case "number":
		_, err := jsonObj.Int()
		if err != nil {
			PrintMessage("❌ Expected number at " + jsonPath)
			return
		}
	case "object":
		obj, err := jsonObj.Object()
		if err != nil {
			PrintMessage("❌ Expected object at " + jsonPath)
			return
		}
		if obj.Len() > len(valNode.Properties) {
			keys := set.New()

			for key := range valNode.Properties {
				keys.Insert(key)
			}

			obj.Visit(func(key []byte, v *fastjson.Value) {
				foundKey := string(key)
				if !keys.Has(foundKey) {
					PrintMessage(fmt.Sprintf("⚠️ Object has unexpected field '%v' at %v", foundKey, jsonPath))
				}
			})

		}
		for propertyName, propertyValNode := range valNode.Properties {
			propertyJsonObj := obj.Get(propertyName)

			if propertyJsonObj == nil {
				if !propertyValNode.Optional {
					PrintMessage(fmt.Sprintf("❌ Object missing key '%v' at %v", propertyName, jsonPath))
				}
				continue
			}

			ValidateJsonFile(*propertyValNode, propertyJsonObj, jsonPath+"."+propertyName, mergedDefinition)
		}
	case "list":
		arr, err := jsonObj.Array()
		if err != nil {
			PrintMessage("❌ Expected list at " + jsonPath)
			return
		}
		for i, childJsonObj := range arr {
			ValidateJsonFile(*valNode.Children, childJsonObj, jsonPath+"["+fmt.Sprint(i)+"]", mergedDefinition)
		}

	default:
		nextValidationNode := mergedDefinition[valNode.Type]

		if nextValidationNode == nil {
			PrintMessage("❗ Unknown type specified at " + jsonPath)
			return
		}

		ValidateJsonFile(*nextValidationNode, jsonObj, jsonPath, mergedDefinition)
	}

}

func main() {
	FailOnError := func(err error) {
		if err != nil {
			log.Fatal(err)
		}
	}

	var directoryPath string
	flag.StringVar(&directoryPath, "directory", ".", "Directory with json files to validate")
	flag.Parse()

	FindTypedefFiles(directoryPath, func(filePath string) {
		typedefFile, err := os.ReadFile(filePath)
		FailOnError(err)

		obj, err := ParseTypedefToNode(typedefFile)
		FailOnError(err)

		FindJsonFiles(filePath, func(filePath string) {
			jsonFile, err := os.ReadFile(filePath)
			FailOnError(err)

			v, err := fastjson.Parse(string(jsonFile))
			FailOnError(err)

			PrintMessage(fmt.Sprintf("📜 Validating %v", filePath))
			ValidateJsonFile(obj, v, "", make(map[string]*Node))
			PrintMessage(fmt.Sprintf("✅ Successfully validated %v\n", filePath))
		})

	})
}
