package cfg

type Config map[string]string

func MakeConfig() Config {
	return make(map[string]string)
}
