package main

import (
	"config-generator/models"
	"encoding/json"
	"github.com/labstack/echo"
	"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	deploymentName  = "deployment"
	serviceName     = "service"
	ingressName     = "ingress"
	baseHelmFolder  = "helm"
	templatesFolder = "templates"
)

var templatesFolderPath = baseHelmFolder + "/" + templatesFolder

func main() {
	e := echo.New()
	e.POST("/configure", GenerateConfigs)
	e.Logger.Fatal(e.Start(":8080"))
}

func GenerateConfigs(c echo.Context) error {
	configs := make(map[string]interface{})

	// reads the JSON object and stores in the map
	err := json.NewDecoder(c.Request().Body).Decode(&configs)
	if err != nil {
		log.Fatal(err)
	}

	// generates values.yaml file from the map
	generateValuesYamlFile(configs)

	// generates helm charts
	configureHelmChart(configs)

	return c.JSON(http.StatusOK, configs)
}

func generateValuesYamlFile(configs map[string]interface{}) {

	// create a helm folder
	err := os.MkdirAll(baseHelmFolder, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	//creates values.yaml file inside the helm folder
	valuesFile, err := os.Create(baseHelmFolder + "/values.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// converts map to yaml file
	yamlData, _ := yaml.Marshal(&configs)
	_, err = valuesFile.Write(yamlData)
	if err != nil {
		log.Fatal(err)
	}
}

func configureHelmChart(configs map[string]interface{}) {

	// temporarily hold the response map in the generatePlaceholders function
	responseMap := make(map[string]interface{})

	// creates templates folder inside helm folder
	err := os.MkdirAll(templatesFolderPath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range configs {

		if strings.EqualFold(key, deploymentName) {
			deploymentObject := models.KubeComponent{
				KubeComponentType: "deployment",
				APIVersion:        "apps/v1",
				Kind:              "Deployment",
				KubeObjectValue:   value,
				KubeObjectKey:     key,
			}
			generateHelmChart(deploymentObject, responseMap)

		} else if strings.EqualFold(key, serviceName) {
			serviceObject := models.KubeComponent{
				KubeComponentType: "service",
				APIVersion:        "v1",
				Kind:              "Service",
				KubeObjectValue:   value,
				KubeObjectKey:     key,
			}
			generateHelmChart(serviceObject, responseMap)

		} else if strings.EqualFold(key, ingressName) {
			ingressObject := models.KubeComponent{
				KubeComponentType: "ingress",
				APIVersion:        "networking.k8s.io/v1",
				Kind:              "Ingress",
				KubeObjectValue:   value,
				KubeObjectKey:     key,
			}
			generateHelmChart(ingressObject, responseMap)
		}
	}
}

func generateHelmChart(kubeObj models.KubeComponent, responseMap map[string]interface{}) {
	// creates deployment.yaml file inside the helm/templates folder
	yamlFile, _ := os.Create(templatesFolderPath + "/" + kubeObj.KubeComponentType + ".yaml")

	// catches the deployment map inside the configs map
	var kubeObjectMap = kubeObj.KubeObjectValue.(map[string]interface{})

	// generates the path placeholder in the deployment helm chart
	generatedKubeObject := generatePlaceholders(kubeObjectMap, responseMap, "", kubeObj.KubeObjectKey)
	generatedKubeObject["apiVersion"] = kubeObj.APIVersion
	generatedKubeObject["kind"] = kubeObj.Kind

	// writes the deployment object to deployment.yaml file
	yamlData, _ := yaml.Marshal(&generatedKubeObject)
	_, err := yamlFile.Write(yamlData)
	if err != nil {
		log.Fatal(err)
	}
}

func generatePlaceholders(obj map[string]interface{}, responseObj map[string]interface{}, extraKey string, typeKey string) map[string]interface{} {
	for k, v := range obj {
		valueMap, isValueMap := v.(map[string]interface{}) // checks if v is of type map
		valueSlice, isValueSlice := v.([]interface{})      // checks if v is of type slice

		if isValueMap { // if v is a map recurse
			generatePlaceholders(valueMap, responseObj, extraKey+k+".", typeKey)
		} else if isValueSlice { // if v is a slice
			for _, s := range valueSlice { // iterates over the slice
				sliceMap, isSliceMap := s.(map[string]interface{}) // checks if s is a map
				if isSliceMap {                                    // if s is a map recurse
					generatePlaceholders(sliceMap, responseObj, extraKey+k+".", typeKey)
				}
			}
		} else { // if v is a leaf node set value for keys
			responseObj[k] = "{{.Values." + typeKey + "." + extraKey + k + "}}" // this temporarily holds the path
			obj[k] = "{{.Values." + typeKey + "." + extraKey + k + "}}"
		}
	}

	return obj
}
