package history

type Storage interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	Data() map[string]any
}
