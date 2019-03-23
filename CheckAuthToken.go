package main

// Copyright (C) Philip Schlump 2018-2019.

import (
	"fmt"
	"net/http"

	"github.com/pschlump/MiscLib"
	"github.com/pschlump/godebug"
)

// CheckAuthToken looks at either a header or a cookie to determine if the user is authorized.
func CheckAuthToken(www http.ResponseWriter, req *http.Request) bool {
	if db_flag["db-auth"] {
		fmt.Printf("In CheckAuthToken: Looking for [%s]\n", gCfg.AuthKey)
	}
	if gCfg.AuthKey == "-none-" {
		if db_flag["db-auth"] {
			fmt.Fprintf(logFilePtr, "%sAuth Success - no authentication%s\n", MiscLib.ColorGreen, MiscLib.ColorReset)
		}
		return true
	}

	// look for cookie
	cookie, err := req.Cookie("Qr-Auth")
	if db_flag["db-auth"] {
		fmt.Printf("Cookie: %s\n", godebug.SVarI(cookie))
	}
	if err == nil {
		if cookie.Value == gCfg.AuthKey {
			if db_flag["db-auth"] {
				fmt.Fprintf(logFilePtr, "%sAuth Success - cookie%s\n", MiscLib.ColorGreen, MiscLib.ColorReset)
			}
			return true
		}
	}

	// look for header
	// ua := r.Header.Get("User-Agent")
	auth := req.Header.Get("X-Qr-Auth")
	if db_flag["db-auth"] {
		fmt.Printf("Header: %s\n", godebug.SVarI(auth))
	}
	if auth == gCfg.AuthKey {
		if db_flag["db-auth"] {
			fmt.Fprintf(logFilePtr, "%sAuth Success - header%s\n", MiscLib.ColorGreen, MiscLib.ColorReset)
		}
		return true
	}

	auth_key_found, auth_key := GetVar("auth_key", www, req)
	if db_flag["db-auth"] {
		fmt.Printf("Variable: %s\n", auth_key)
	}
	if auth_key_found && auth_key == gCfg.AuthKey {
		if db_flag["db-auth"] {
			fmt.Fprintf(logFilePtr, "%sAuth Success - header%s\n", MiscLib.ColorGreen, MiscLib.ColorReset)
		}
		return true
	}

	if db_flag["db-auth"] {
		fmt.Fprintf(logFilePtr, "%sAuth Fail%s\n", MiscLib.ColorRed, MiscLib.ColorReset)
	}
	return false
}
