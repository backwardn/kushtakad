package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"

	"github.com/kushtaka/kushtakad/helpers"
)

const StaticUrl = "http://abcdefghijklmnopqrstuvwxyz.zyxwvutsrqponmlkjihgfedcba.aceegikmoqsuwy.bdfhjlnprtvxz"
const StreamOffset = 793

func assertSize(name string, want, is int) error {
	if want != is {
		e := fmt.Sprintf("assertSize() %s offset wants:%d != is:%d", name, want, is)
		return errors.New(e)
	}
	return nil
}

type PdfContext struct {
	Key      string
	Url      string
	FileByes []byte
}

func BuildPdf(url string, pdfb []byte) (pdfc *PdfContext, err error) {
	var bf bytes.Buffer
	var size int
	var bytesZlib []byte
	var retKey, retUrl string
	pdfc = &PdfContext{}
	size = 1

	re := regexp.MustCompile(".*\\/Length ([0-9]+)\\/.*")
	match := re.FindAll(pdfb[StreamOffset:], -1)

	lineLength := len(match[0]) - len("stream\n")
	if err := assertSize("lineLength", 67, lineLength); err != nil {
		return nil, err
	}

	stream_beg := StreamOffset + lineLength
	if err := assertSize("stream_beg", 860, stream_beg); err != nil {
		return nil, err
	}

	stream_start := stream_beg + 8
	if err := assertSize("stream_start", 868, stream_start); err != nil {
		return nil, err
	}

	stream_header := pdfb[StreamOffset:stream_start]

	submatch := re.FindAllSubmatch(pdfb[StreamOffset:], -1)
	s := fmt.Sprintf("%s", submatch[0][1])
	stream_size, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}

	if err := assertSize("stream_size", 110, stream_size); err != nil {
		return nil, err
	}

	stream := pdfb[stream_start : stream_start+stream_size]
	if err := assertSize("stream", 110, len(stream)); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	newreader := bytes.NewReader(stream)
	zr, err := zlib.NewReader(newreader)
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(buf, zr)
	zr.Close()

	// this brute forces a url of the correct  buffer size in order to fit in the zlib compressed space of the pdf
	for len(bytesZlib) != len(buf.Bytes()) {
		retUrl, retKey = helpers.GenerateLink(url, "t", size)
		rp := bytes.Replace([]byte(buf.String()), []byte(StaticUrl), []byte(retUrl), -1)

		var bf bytes.Buffer
		zw, err := zlib.NewWriterLevel(&bf, 6)
		if err != nil {
			log.Fatal(err)
		}
		_, err = zw.Write([]byte(rp))
		if err != nil {
			log.Fatal(err)
		}
		zw.Close()

		bytesZlib = bf.Bytes()
		if len(bytesZlib) > len(buf.Bytes()) {
			e := `
			This is a hacky bruteforce method. We will try and fix it later and you shouldn't see this error but... 
			The zlib compressed URL we are replacing is too large and won't fit in the pdf. 
			That shouldn't be the case, BUT somehow you broke it. Maybe you are using a HUGE domain name?
			I'd like to understand more so feel free to file a bug project @ https://github.com/kushtaka/kushtakad.
			`
			return nil, errors.New(e)
		}
		pdfc.Key = retKey
		pdfc.Url = retUrl
		size = size + 1
	}

	bw := bufio.NewWriter(&bf)

	_, err = bw.Write(pdfb[0:StreamOffset])
	if err != nil {
		return nil, err
	}

	err = bw.Flush()
	if err != nil {
		return nil, err
	}

	_, err = bw.Write(stream_header)
	if err != nil {
		return nil, err
	}

	err = bw.Flush()
	if err != nil {
		return nil, err
	}

	_, err = bw.Write(bytesZlib)
	if err != nil {
		return nil, err
	}

	err = bw.Flush()
	if err != nil {
		return nil, err
	}

	_, err = bw.Write(pdfb[stream_start+stream_size:])
	if err != nil {
		return nil, err
	}

	err = bw.Flush()
	if err != nil {
		return nil, err
	}

	pdfc.FileByes = bf.Bytes()
	return pdfc, nil

}
