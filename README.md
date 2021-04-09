# Helm Images

Basic flow:
1. Given a helm chart, get a list of all possible images.
   chart ->
```json
[
  "{{ .Values.chrome.image }}:{{ .Values.chrome.tag }}",
  "{{ .Values.chromeDebug.image }}:{{ .Values.chromeDebug.tag }}",
  "{{ .Values.firefox.image }}:{{ .Values.firefox.tag }}",
  "{{ .Values.firefoxDebug.image }}:{{ .Values.firefoxDebug.tag }}",
  "{{ .Values.hub.image }}:{{ .Values.hub.tag }}"
]
```
2. For each of those images, render with the values to find the source image
    ->
```json
[
  "selenium/node-chrome:3.141.59",
  "selenium/node-chrome-debug:3.141.59",
  "selenium/node-firefox:3.141.59",
  "selenium/node-firefox-debug:3.141.59",
  "selenium/hub:3.141.59"
]
```

3. Parse the image format
4. Find any parts that correspond to the registry
5. Re-write with new registry