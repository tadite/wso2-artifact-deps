Tool for analyse of wso2 esb 4.9.0 carbon-apps dependencies. 

Parses artifact.xml and find all occurances of artifacts in xml files of other wso2 artifacts.

Output carbon-apps dependencies graph in .png and .dot

#### args
-path - path to root dit with carbon-apps to analyse (if absent, execution path will be used)

-outPath - path to save graph (if absent, execution path will be used)

-carsToAnalyse - list of carbon-apps names to draw dependencies graph (if absent, all car-apps deps will be rendered)

-ignoreCarRegex - regex for ignoring car-app names during analysis

```
artifact-deps.exe -path="D:\car-apps-root" -outPath="D:\deps-result" -carsToAnalyse="carname1, carname2 -ignoreCarRegex=".+STUB.+|.+Common.+"
```
