package model

import(

)

type Fields struct {}

func setFilds(parametrs map) {
	for nameField, typeField := range parametrs {
		Fields := {
			nameField	typeField
		}
	}
}

func (f *Fields) getFields() (struct){
	return f
}



