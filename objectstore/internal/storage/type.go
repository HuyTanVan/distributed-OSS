package storage

type ObjectMeta struct {
	Bucket string
	Key    string
	Hash   string
	Size   int64
}
