package main

type config map[string]string

func makeConfig() config {
	return make(map[string]string)
}
