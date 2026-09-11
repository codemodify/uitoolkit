package fonts

import "testing"

func TestEmbeddedFacesLoad(t *testing.T) {
	for _, name := range []string{
		FileTitilliumRegular, FileTitilliumBold,
		FileJetBrainsRegular, FileJetBrainsBold,
	} {
		b, err := Bytes(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(b) < 1024 {
			t.Fatalf("%s: %d bytes is not a TTF", name, len(b))
		}
		if string(b[:4]) != "\x00\x01\x00\x00" && string(b[:4]) != "OTTO" {
			t.Fatalf("%s: not a TTF/OTF header %q", name, b[:4])
		}
	}
	if FamilyUI != "Titillium Web" || FamilyMono != "JetBrains Mono" {
		t.Fatalf("family constants %q %q", FamilyUI, FamilyMono)
	}
}
