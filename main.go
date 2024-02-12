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
}

func ParseTypedefToNode(typedefFile []byte) (Node, error) {
	var obj Node
	err := json.Unmarshal(typedefFile, &obj)
	return obj, err
}

func PrintMessage(message string) {
	fmt.Println("[JTC]: " + message)
}

func ValidateJsonFile(valNode Node, jsonObj *fastjson.Value, jsonPath string) {
	switch valNode.Type {
	case "string":
		_, err := jsonObj.StringBytes()
		if err != nil {
			PrintMessage("❌ Expected string at " + jsonPath)
		}
	case "number":
		_, err := jsonObj.Int()
		if err != nil {
			PrintMessage("❌ Expected number at " + jsonPath)
		}
	case "object":
		obj, err := jsonObj.Object()
		if err != nil {
			PrintMessage("❌ Expected object at " + jsonPath)
		}
		if obj.Len() != len(valNode.Properties) {
			PrintMessage("⚠️ Object has too many fields at " + jsonPath)
		}
		for propertyName, propertyValNode := range valNode.Properties {
			propertyJsonObj := obj.Get(propertyName)
			if propertyJsonObj == nil {
				PrintMessage(fmt.Sprintf("❌ Object missing key '%v' at %v", propertyName, jsonPath))
			} else {

				ValidateJsonFile(*propertyValNode, propertyJsonObj, jsonPath+"."+propertyName)
			}
		}
	case "list":
		arr, err := jsonObj.Array()
		if err != nil {
			PrintMessage("❌ Expected list at " + jsonPath)
		}
		for i, childJsonObj := range arr {
			ValidateJsonFile(*valNode.Children, childJsonObj, jsonPath+"["+fmt.Sprint(i)+"]")
		}

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
			ValidateJsonFile(obj, v, "")
			PrintMessage(fmt.Sprintf("✅ Successfully validated %v\n", filePath))
		})

	})
}
