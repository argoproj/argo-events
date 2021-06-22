package main

import (
	"encoding/json"
	"os"
)

func kubeifySwagger(in, out string) {
	f, err := os.Open(in)
	if err != nil {
		panic(err)
	}
	swagger := obj{}
	err = json.NewDecoder(f).Decode(&swagger)
	if err != nil {
		panic(err)
	}
	definitions := swagger["definitions"].(obj)

	for _, d := range definitions {
		props, ok := d.(obj)["properties"].(obj)
		if ok {
			for _, prop := range props {
				prop := prop.(obj)
				// if prop["format"] == "int32" || prop["format"] == "int64" {
				// 	delete(prop, "format")
				// }
				delete(prop, "default")
				items, ok := prop["items"].(obj)
				if ok {
					delete(items, "default")
				}
				additionalProperties, ok := prop["additionalProperties"].(obj)
				if ok {
					delete(additionalProperties, "default")
				}
			}
		}
		props, ok = d.(obj)["additionalProperties"].(obj)
		if ok {
			delete(props, "default")
		}
	}

	f, err = os.Create(out)
	if err != nil {
		panic(err)
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	err = e.Encode(swagger)
	if err != nil {
		panic(err)
	}
	err = f.Close()
	if err != nil {
		panic(err)
	}
}
