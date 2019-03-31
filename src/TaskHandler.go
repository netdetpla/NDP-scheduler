package main

type param struct {
	Parameter string `json:"parameter"`
}

type dispatch struct {
	ImageName string `json:"image_name"`
	ParamList []param `json:"parameter_list"`
}

var taskQueue = make(chan taskInfo, 50)

