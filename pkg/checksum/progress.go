package checksum

type ProgressFunc func(ProgressEvent)

// "marker event"
type ProgressEvent interface {
	isProgressEvent()
}

type MostCurrentFoundFile struct{ Path string }

func (MostCurrentFoundFile) isProgressEvent() {}

type MostCurrentIgnoredPath struct{ Path string }

func (MostCurrentIgnoredPath) isProgressEvent() {}

type MostCurrentMergeHashFile struct{ Path string }

func (MostCurrentMergeHashFile) isProgressEvent() {}

type DiscoverFilesFound struct{ Count uint64 }

func (DiscoverFilesFound) isProgressEvent() {}

type DiscoverFilesIgnored struct{ Path string }

func (DiscoverFilesIgnored) isProgressEvent() {}

type DiscoverFilesDone struct {
	Found   uint64
	Ignored uint64
}

func (DiscoverFilesDone) isProgressEvent() {}

type PreRead struct{ Path string }

func (PreRead) isProgressEvent() {}

type ReadProgress struct {
	Read  uint64
	Total uint64
}

func (ReadProgress) isProgressEvent() {}

type FileMatch struct{ Path string }

func (FileMatch) isProgressEvent() {}

type FileUnchangedSkipped struct{ Path string }

func (FileUnchangedSkipped) isProgressEvent() {}

type FileChanged struct{ Path string }

func (FileChanged) isProgressEvent() {}

type FileChangedCorrupted struct{ Path string }

func (FileChangedCorrupted) isProgressEvent() {}

type FileChangedOlder struct{ Path string }

func (FileChangedOlder) isProgressEvent() {}

type FileNew struct{ Path string }

func (FileNew) isProgressEvent() {}

type FileRemoved struct{ Path string }

func (FileRemoved) isProgressEvent() {}

type Finished struct{}

func (Finished) isProgressEvent() {}
