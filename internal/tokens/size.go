package tokens

const (
	sizeThresholdTiny   = int64(1 << 20)         // 1 MB
	sizeThresholdSmall  = int64(100 * (1 << 20)) // 100 MB
	sizeThresholdMedium = int64(1 << 30)         // 1 GB
)

func fileSizeRange(size int64) string {
	switch {
	case size < sizeThresholdTiny:
		return "tiny"
	case size < sizeThresholdSmall:
		return "small"
	case size < sizeThresholdMedium:
		return "medium"
	default:
		return "large"
	}
}
