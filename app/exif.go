package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// exifInfo is the metadata the property panel shows, read from a JPEG's
// APP1 EXIF segment.
type exifInfo struct {
	DateTimeOriginal time.Time
	Make, Model      string
	Orientation      int
	ISO              int
	FNumber          float64
	ExposureTime     string
	FocalLength      int
	HasGPS           bool
	Lat, Lon         float64
}

// readEXIF reads EXIF from a JPEG's bytes. Other formats get an empty value.
func readEXIF(data []byte) exifInfo {
	var info exifInfo
	tiff, ok := exifSegment(data)
	if !ok {
		return info
	}
	ifd, order, ok := tiffHeader(tiff)
	if !ok {
		return info
	}
	tags := readIFD(tiff, ifd, order, 0)
	if v, ok := tags[0x0112]; ok {
		info.Orientation = int(toUint(v, order))
	}
	info.Make = toString(tags[0x010F], order)
	info.Model = toString(tags[0x0110], order)

	if exifOff, ok := tags[0x8769]; ok {
		sub := readIFD(tiff, int(toUint(exifOff, order)), order, 0)
		if v, ok := sub[0x9003]; ok {
			info.DateTimeOriginal = parseEXIFTime(toString(v, order))
		} else if v, ok := sub[0x0132]; ok {
			info.DateTimeOriginal = parseEXIFTime(toString(v, order))
		}
		if v, ok := sub[0x8827]; ok {
			info.ISO = int(toUint(v, order))
		}
		if v, ok := sub[0x829D]; ok {
			info.FNumber = toRational(v, order)
		}
		if v, ok := sub[0x829A]; ok {
			num, den := toRationalParts(v, order)
			if num > 0 && den > 0 {
				if num < den {
					info.ExposureTime = fmt.Sprintf("1/%d s", int(float64(den)/float64(num)+0.5))
				} else {
					info.ExposureTime = fmt.Sprintf("%.1f s", float64(num)/float64(den))
				}
			}
		}
		if v, ok := sub[0x920A]; ok {
			info.FocalLength = int(toRational(v, order) + 0.5)
		}
	}
	if gpsOff, ok := tags[0x8825]; ok {
		gps := readIFD(tiff, int(toUint(gpsOff, order)), order, 0)
		lat, okLat := gpsCoord(gps[2], gps[1], order)
		lon, okLon := gpsCoord(gps[4], gps[3], order)
		if okLat && okLon {
			info.HasGPS, info.Lat, info.Lon = true, lat, lon
		}
	}
	return info
}

// exifSegment returns the TIFF part of the first EXIF APP1 segment.
func exifSegment(data []byte) ([]byte, bool) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, false
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return nil, false
		}
		marker := data[i+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if marker == 0xDA {
			break // image data starts
		}
		size := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		if size < 2 || i+2+size > len(data) {
			return nil, false
		}
		payload := data[i+4 : i+2+size]
		if marker == 0xE1 && len(payload) > 6 && string(payload[:6]) == "Exif\x00\x00" {
			return payload[6:], true
		}
		i += 2 + size
	}
	return nil, false
}

// tiffHeader reads the TIFF header and returns the offset of the first IFD.
func tiffHeader(tiff []byte) (int, binary.ByteOrder, bool) {
	if len(tiff) < 8 {
		return 0, nil, false
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0, nil, false
	}
	return int(order.Uint32(tiff[4:8])), order, true
}

// readIFD reads the tags of one IFD.
func readIFD(tiff []byte, off int, order binary.ByteOrder, depth int) map[uint16][]byte {
	out := map[uint16][]byte{}
	if depth > 4 || off <= 0 || off+2 > len(tiff) {
		return out
	}
	count := int(order.Uint16(tiff[off : off+2]))
	for i := 0; i < count; i++ {
		entry := off + 2 + i*12
		if entry+12 > len(tiff) {
			return out
		}
		tag := order.Uint16(tiff[entry : entry+2])
		typ := order.Uint16(tiff[entry+2 : entry+4])
		n := int(order.Uint32(tiff[entry+4 : entry+8]))
		unit := typeSize(typ)
		if unit == 0 {
			continue
		}
		total := unit * n
		var value []byte
		if total <= 4 {
			value = tiff[entry+8 : entry+12]
		} else {
			ptr := int(order.Uint32(tiff[entry+8 : entry+12]))
			if ptr < 0 || ptr+total > len(tiff) {
				continue
			}
			value = tiff[ptr : ptr+total]
		}
		out[tag] = value[:min(total, len(value))]
	}
	return out
}

// typeSize is the size in bytes of one EXIF value of type typ.
func typeSize(typ uint16) int {
	switch typ {
	case 1, 2, 6, 7:
		return 1
	case 3, 8:
		return 2
	case 4, 9, 11:
		return 4
	case 5, 10, 12:
		return 8
	}
	return 0
}

// toUint reads a short or long value.
func toUint(v []byte, order binary.ByteOrder) uint32 {
	switch len(v) {
	case 2:
		return uint32(order.Uint16(v))
	case 4:
		return order.Uint32(v)
	}
	return 0
}

// toString reads a text value.
func toString(v []byte, order binary.ByteOrder) string {
	if len(v) == 0 {
		return ""
	}
	s := string(bytes.TrimRight(v, "\x00"))
	return strings.TrimSpace(s)
}

// toRationalParts reads a rational value.
func toRationalParts(v []byte, order binary.ByteOrder) (uint32, uint32) {
	if len(v) < 8 {
		return 0, 0
	}
	return order.Uint32(v[:4]), order.Uint32(v[4:8])
}

// toRational reads a rational value as a number.
func toRational(v []byte, order binary.ByteOrder) float64 {
	num, den := toRationalParts(v, order)
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// gpsCoord turns a GPS coordinate and its reference into degrees.
func gpsCoord(value, ref []byte, order binary.ByteOrder) (float64, bool) {
	if len(value) < 24 {
		return 0, false
	}
	d := toRational(value[0:8], order)
	m := toRational(value[8:16], order)
	s := toRational(value[16:24], order)
	out := d + m/60 + s/3600
	if r := strings.ToUpper(toString(ref, order)); r == "S" || r == "W" {
		out = -out
	}
	return out, true
}

// parseEXIFTime parses the "2006:01:02 15:04:05" EXIF format.
func parseEXIFTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if len(s) < 19 {
		return time.Time{}
	}
	t, err := time.Parse("2006:01:02 15:04:05", s[:19])
	if err != nil {
		return time.Time{}
	}
	return t
}

// xmpHeader marks an XMP APP1 segment.
const xmpHeader = "http://ns.adobe.com/xap/1.0/\x00"

// writeMetadata stores the caption, the tags and the star rating in the
// photo's own metadata: an XMP packet in a JPEG, and an XMP sidecar next to
// any other kind of file. The library keeps its own copy either way.
func writeMetadata(path, caption string, tags []string, stars int) error {
	packet := xmpPacket(caption, tags, stars)
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" {
		return os.WriteFile(sidecarPath(path), []byte(packet), 0o644)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return errors.New("not a JPEG file")
	}
	segment := xmpSegment(packet)
	out, err := replaceXMPSegment(data, segment)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".metadata-*.jpg")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}

// sidecarPath is where metadata of a non-JPEG photo goes.
func sidecarPath(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".xmp"
}

// xmpPacket builds an XMP packet with a title, keywords and a rating.
func xmpPacket(caption string, tags []string, stars int) string {
	var b strings.Builder
	b.WriteString(`<?xpacket begin="` + "\uFEFF" + `" id="W5M0MpCehiHzreSzNTczkc9d"?>` + "\n")
	b.WriteString(`<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Toki">` + "\n")
	b.WriteString(` <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` + "\n")
	b.WriteString(`  <rdf:Description rdf:about="" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:xmp="http://ns.adobe.com/xap/1.0/">` + "\n")
	if caption != "" {
		b.WriteString(`   <dc:title><rdf:Alt><rdf:li xml:lang="x-default">` + xmlEscape(caption) + `</rdf:li></rdf:Alt></dc:title>` + "\n")
	}
	if len(tags) > 0 {
		b.WriteString(`   <dc:subject><rdf:Bag>`)
		for _, t := range tags {
			b.WriteString(`<rdf:li>` + xmlEscape(t) + `</rdf:li>`)
		}
		b.WriteString(`</rdf:Bag></dc:subject>` + "\n")
	}
	b.WriteString(`   <xmp:Rating>` + strconv.Itoa(stars) + `</xmp:Rating>` + "\n")
	b.WriteString(`  </rdf:Description>` + "\n")
	b.WriteString(` </rdf:RDF>` + "\n")
	b.WriteString(`</x:xmpmeta>` + "\n")
	b.WriteString(`<?xpacket end="w"?>`)
	return b.String()
}

// xmpSegment wraps an XMP packet in an APP1 segment.
func xmpSegment(packet string) []byte {
	payload := append([]byte(xmpHeader), []byte(packet)...)
	size := len(payload) + 2
	if size > 0xFFFF {
		payload = payload[:0xFFFF-2]
		size = len(payload) + 2
	}
	out := make([]byte, 0, size+2)
	out = append(out, 0xFF, 0xE1, byte(size>>8), byte(size))
	return append(out, payload...)
}

// replaceXMPSegment swaps the first XMP APP1 segment for segment, or inserts
// it right after the start-of-image marker.
func replaceXMPSegment(jpeg, segment []byte) ([]byte, error) {
	i := 2
	for i+4 <= len(jpeg) {
		if jpeg[i] != 0xFF {
			return nil, errors.New("malformed JPEG")
		}
		marker := jpeg[i+1]
		if marker == 0xDA {
			break
		}
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		size := int(binary.BigEndian.Uint16(jpeg[i+2 : i+4]))
		if size < 2 || i+2+size > len(jpeg) {
			return nil, errors.New("malformed JPEG segment")
		}
		payload := jpeg[i+4 : i+2+size]
		if marker == 0xE1 && len(payload) > len(xmpHeader) && string(payload[:len(xmpHeader)]) == xmpHeader {
			out := make([]byte, 0, len(jpeg)+len(segment))
			out = append(out, jpeg[:i]...)
			out = append(out, segment...)
			return append(out, jpeg[i+2+size:]...), nil
		}
		i += 2 + size
	}
	out := make([]byte, 0, len(jpeg)+len(segment))
	out = append(out, jpeg[:2]...)
	out = append(out, segment...)
	return append(out, jpeg[2:]...), nil
}

// xmlEscape escapes the characters XML forbids in text.
func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
