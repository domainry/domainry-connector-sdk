package httpapi

import (
	"reflect"
	"testing"
)

func TestDocumentFormatsMatchBuilderRoutes(t *testing.T) {
	want := []string{
		".pdf", ".docx", ".pptx", ".xlsx", ".xlsm",
		".txt", ".md", ".rst", ".html", ".htm", ".csv", ".json", ".jsonl",
		".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".go", ".rs", ".c", ".h", ".cpp", ".hpp", ".cs", ".rb", ".php", ".kt", ".swift", ".scala", ".sh",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp",
		".mp3", ".wav", ".ogg", ".flac", ".m4a", ".aac",
		".mp4", ".mov", ".webm", ".avi",
	}
	formats := DocumentFormatCapabilities()
	got := make([]string, 0, len(formats))
	for _, format := range formats {
		got = append(got, format.Extension)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extensions=%v", got)
	}
	for _, filename := range []string{"legacy.doc", "legacy.ppt", "legacy.xls", "audio.opus", "table.tsv", "video.mkv"} {
		if _, ok := DocumentFormat(filename); ok || MaximumInlineDocumentBytesForFilename(filename) != 0 || BuilderWholeFileMaximumBytesForFilename(filename) != 0 {
			t.Fatalf("unsupported Builder route accepted: %s", filename)
		}
	}
	media, ok := DocumentFormat("meeting.mp4")
	if !ok || media.BuilderWholeFileMaximumBytes != BuilderMediaMaximumBytes || media.EnrichmentMaxBytes != 0 || MaximumInlineDocumentBytesForFilename("meeting.mp4") != MaximumInlineDocumentBytes || BuilderWholeFileMaximumBytesForFilename("meeting.mp4") != BuilderMediaMaximumBytes {
		t.Fatalf("media=%+v ok=%v", media, ok)
	}
	image, ok := DocumentFormat("diagram.png")
	if !ok || image.BuilderWholeFileMaximumBytes != 0 || image.AssetMaximumBytes != BuilderAssetMaximumBytes || image.EnrichmentMaxBytes != BuilderImageEnrichmentMaximumBytes || BuilderWholeFileMaximumBytesForFilename("scan.png") != 0 {
		t.Fatalf("image=%+v ok=%v", image, ok)
	}
	table, ok := DocumentFormat("data.jsonl")
	if !ok || table.AnalysisMaxBytes != BuilderAnalysisOutputMaximumBytes {
		t.Fatalf("structured=%+v ok=%v", table, ok)
	}
}
