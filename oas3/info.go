package oas3

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jayvynl/goctl-openapi/constant"
)

func getInfo(properties map[string]string) *openapi3.Info {
	title := GetProperty(properties, constant.ApiInfoTitle)
	version := GetProperty(properties, constant.ApiInfoVersion)
	desc := GetProperty(properties, constant.ApiInfoDesc)
	author := GetProperty(properties, constant.ApiInfoAuthor)
	email := GetProperty(properties, constant.ApiInfoEmail)
	info := &openapi3.Info{
		Title:       title,
		Description: desc,
		Version:     version,
	}
	if author != "" || email != "" {
		info.Contact = &openapi3.Contact{
			Name:  author,
			Email: email,
		}
	}
	return info
}

func getServers(properties map[string]string) openapi3.Servers {
	urls := GetProperty(properties, constant.ApiInfoServers)
	urls = strings.TrimSpace(urls)
	if urls == "" {
		return nil
	}

	urlList := strings.Split(urls, ",")
	servers := make(openapi3.Servers, len(urlList))
	for i, url := range urlList {
		servers[i] = &openapi3.Server{URL: url}
	}
	return servers
}

func getExternalDocs(properties map[string]string) *openapi3.ExternalDocs {
	url := GetProperty(properties, constant.ApiInfoExternalDocs)
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	return &openapi3.ExternalDocs{
		URL: url,
	}
}

func getTags(properties map[string]string) []string {
	names := GetProperty(properties, constant.ApiInfoTags)
	names = strings.TrimSpace(names)
	if names == "" {
		return nil
	}

	return strings.Split(names, ",")
}
