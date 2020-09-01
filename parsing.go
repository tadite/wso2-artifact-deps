package main

import (
	"github.com/beevik/etree"
	"log"
	"regexp"
	"strings"
)

func FindArtifactsInDoc(doc *etree.Document) []string {
	var foundArtifacts []string

	childElements := doc.ChildElements()
	rootElementName := childElements[0].Tag
	switch rootElementName {
	case "proxy", "sequence", "template", "api":
		foundArtifacts = append(foundArtifacts, *FindTemplates(doc)...)
		foundArtifacts = append(foundArtifacts, *FindSequences(doc)...)
		foundArtifacts = append(foundArtifacts, *FindResources(doc)...)
		foundArtifacts = append(foundArtifacts, *FindLocalEntriesUseInProperty(doc)...)
	case "task":
		foundArtifacts = append(foundArtifacts, *FindSequenceInTask(doc)...)
	default:
		log.Print(rootElementName)
	}
	return foundArtifacts
}

var getPropertyFuncRegex = regexp.MustCompile("get-property\\('(.+?)'\\)")

func FindLocalEntriesUseInProperty(doc *etree.Document) *[]string {
	var foundArtifacts []string
	propertyElements := doc.FindElements("//property")
	for _, element := range propertyElements {
		expressionAttr := element.SelectAttr("expression")
		if expressionAttr != nil {
			expressionAttrValue := expressionAttr.Value
			foundGetPropertyArgs := getPropertyFuncRegex.FindAllString(expressionAttrValue, -1)
			foundArtifacts = append(foundArtifacts, foundGetPropertyArgs...)
		}
	}
	return &foundArtifacts
}

func FindResources(doc *etree.Document) *[]string {
	var foundArtifacts []string
	resourcesElements := doc.FindElements("//schema")
	resourcesElements = append(resourcesElements, doc.FindElements("//resource")...)
	resourcesElements = append(resourcesElements, doc.FindElements("//xslt")...)
	resourcesElements = append(resourcesElements, doc.FindElements("//publishWSDL")...)
	for _, element := range resourcesElements {
		targetAttr := element.SelectAttr("key")
		if targetAttr != nil {
			targetAttrValue := targetAttr.Value
			targetAttrValue = strings.TrimPrefix(targetAttrValue, "gov:")
			foundArtifacts = append(foundArtifacts, targetAttrValue)
		}
	}
	return &foundArtifacts
}

func FindSequenceInTask(doc *etree.Document) *[]string {
	var foundArtifacts []string
	propertyElement := doc.FindElement("//property[@name='sequenceName']")
	valueAttr := propertyElement.SelectAttr("value")
	if valueAttr != nil {
		foundArtifacts = append(foundArtifacts, valueAttr.Value)
	}
	return &foundArtifacts
}

func FindTemplates(doc *etree.Document) *[]string {
	var foundArtifacts []string
	elements := doc.FindElements("//call-template")
	for _, element := range elements {
		targetAttr := element.SelectAttr("target")
		if targetAttr != nil {
			foundArtifacts = append(foundArtifacts, targetAttr.Value)
		}
	}
	return &foundArtifacts
}

func FindSequences(doc *etree.Document) *[]string {
	var foundArtifacts []string
	elements := doc.FindElements("//sequence")
	for _, element := range elements {
		keyAttr := element.SelectAttr("key")
		if keyAttr != nil {
			foundArtifacts = append(foundArtifacts, keyAttr.Value)
		}
	}
	return &foundArtifacts
}
