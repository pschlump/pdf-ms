package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/pschlump/godebug"
	"github.com/pschlump/ms"
	template "github.com/pschlump/textTemplate"
)

// - check that the template has all necessary named-temlates in it. (Function)
func ValidateTemplateHas(TemplateFn string, nameSet []string) (err error) {
	rtFuncMap := template.FuncMap{
		"Center":      ms.CenterStr,   //
		"PadR":        ms.PadOnRight,  //
		"PadL":        ms.PadOnLeft,   //
		"PicTime":     ms.PicTime,     //
		"FTime":       ms.StrFTime,    //
		"PicFloat":    ms.PicFloat,    //
		"nvl":         ms.Nvl,         //
		"Concat":      ms.Concat,      //
		"title":       strings.Title,  // The name "title" is what the function will be called in the template text.
		"ifDef":       ms.IfDef,       //
		"ifIsDef":     ms.IfIsDef,     //
		"ifIsNotNull": ms.IfIsNotNull, //
	}

	t, err := template.New("simple-tempalte").Funcs(rtFuncMap).ParseFiles(TemplateFn)
	// t, err := template.New("simple-tempalte").ParseFiles(TemplateFn)
	if err != nil {
		fmt.Printf("Error(12004): parsing/reading template, %s, AT:%s\n", err, godebug.LF())
		return fmt.Errorf("Error(12004): parsing/reading template, %s, AT:%s\n", err, godebug.LF())
	}

	has := t.AvailableTemplates()
	if missing, ok := Contains(nameSet, has); !ok {
		return fmt.Errorf("Missing Template Items %s", missing)
	}
	return nil
}

func Contains(lookFor, has []string) (missing []string, allFound bool) {
	allFound = true
	for _, xx := range lookFor {
		if InArray(xx, has) {
		} else {
			allFound = false
			missing = append(missing, xx)
		}
	}
	return
}

func InArray(lookFor string, inArr []string) bool {
	for _, v := range inArr {
		if lookFor == v {
			return true
		}
	}
	return
}

// if n, err = IsANumber ( page, www, req ) ; err != nil {
func IsANumber(s string, www http.ResponseWriter, req *http.Request) (nv int, err error) {
	var nn int64
	nn, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		www.WriteHeader(400) // xyzzy fix to name
	} else {
		nv = int(nn)
	}
	return
}

func IsAuthKeyValid(www http.ResponseWriter, req *http.Request) bool {
	// fmt.Printf("AT: %s - gCfg.AuthKey = [%s]\n", godebug.LF(), gCfg.AuthKey)
	found, auth_key := GetVar("auth_key", www, req)
	if gCfg.AuthKey != "" {
		// fmt.Printf("AT: %s - configed AuthKey [%s], found=%v ?auth_key=[%s]\n", godebug.LF(), gCfg.AuthKey, found, auth_key)
		if !found || auth_key != gCfg.AuthKey {
			// fmt.Printf("AT: %s\n", godebug.LF())
			www.WriteHeader(http.StatusUnauthorized) // 401
			return false
		}
	}
	// fmt.Printf("AT: %s\n", godebug.LF())
	return true
}

// LogFile sets the output log file to an open file.  This will turn on logging of SQL statments.
func LogFile(f *os.File) {
	logFilePtr = f
}
