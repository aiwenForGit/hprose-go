/**********************************************************\
|                                                          |
|                          hprose                          |
|                                                          |
| Official WebSite: http://www.hprose.com/                 |
|                   http://www.hprose.net/                 |
|                   http://www.hprose.org/                 |
|                                                          |
\**********************************************************/
/**********************************************************\
 *                                                        *
 * hprose/raw_reader.go                                   *
 *                                                        *
 * hprose RawReader for Go.                               *
 *                                                        *
 * LastModified: Feb 15, 2014                             *
 * Author: Ma Bingyao <andot@hprose.com>                  *
 *                                                        *
\**********************************************************/

package hprose

import (
	"bytes"
	"errors"
)

type RawReader struct {
	stream BufReader
	strbuf [64]byte
}

func NewRawReader(stream BufReader) *RawReader {
	return &RawReader{stream: stream}
}

func (r *RawReader) ReadRaw() (raw []byte, err error) {
	ostream := new(bytes.Buffer)
	err = r.ReadRawTo(ostream)
	return ostream.Bytes(), err
}

func (r *RawReader) ReadRawTo(ostream BufWriter) (err error) {
	var tag byte
	if tag, err = r.stream.ReadByte(); err == nil {
		err = r.readRaw(ostream, tag)
	}
	return err
}

func (r *RawReader) readRaw(ostream BufWriter, tag byte) (err error) {
	if err = ostream.WriteByte(tag); err == nil {
		switch tag {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
			TagNull, TagEmpty, TagTrue, TagFalse, TagNaN:
		case TagInfinity:
			if tag, err = r.stream.ReadByte(); err == nil {
				err = ostream.WriteByte(tag)
			}
		case TagInteger, TagLong, TagDouble, TagRef:
			err = r.readNumberRaw(ostream)
		case TagDate, TagTime:
			err = r.readDateTimeRaw(ostream)
		case TagUTF8Char:
			err = r.readUTF8CharRaw(ostream)
		case TagBytes:
			err = r.readBytesRaw(ostream)
		case TagString:
			err = r.readStringRaw(ostream)
		case TagGuid:
			err = r.readGuidRaw(ostream)
		case TagList, TagMap, TagObject:
			err = r.readComplexRaw(ostream)
		case TagClass:
			if err = r.readComplexRaw(ostream); err == nil {
				err = r.ReadRawTo(ostream)
			}
		case TagError:
			err = r.ReadRawTo(ostream)
		default:
			err = unexpectedTag(tag, nil)
		}
	}
	return err
}

func (r *RawReader) readNumberRaw(ostream BufWriter) (err error) {
	for err == nil {
		var tag byte
		if tag, err = r.stream.ReadByte(); err == nil {
			if err = ostream.WriteByte(tag); tag == TagSemicolon {
				break
			}
		}
	}
	return err
}

func (r *RawReader) readDateTimeRaw(ostream BufWriter) (err error) {
	for err == nil {
		var tag byte
		if tag, err = r.stream.ReadByte(); err == nil {
			if err = ostream.WriteByte(tag); tag == TagSemicolon || tag == TagUTC {
				break
			}
		}
	}
	return err
}

func (r *RawReader) readUTF8CharRaw(ostream BufWriter) (err error) {
	var c rune
	if c, _, err = r.stream.ReadRune(); err == nil {
		_, err = ostream.WriteRune(c)
	}
	return err
}

func (r *RawReader) readBytesRaw(ostream BufWriter) (err error) {
	count := 0
	tag := byte('0')
	for err == nil {
		count *= 10
		count += int(tag - '0')
		if tag, err = r.stream.ReadByte(); err == nil {
			if err = ostream.WriteByte(tag); tag == TagQuote {
				break
			}
		}
	}
	if err == nil {
		b := make([]byte, count+1)
		if _, err = r.stream.Read(b); err == nil {
			_, err = ostream.Write(b)
		}
	}
	return err
}

func (r *RawReader) readStringRaw(ostream BufWriter) (err error) {
	count := 0
	tag := byte('0')
	for err == nil {
		count *= 10
		count += int(tag - '0')
		if tag, err = r.stream.ReadByte(); err == nil {
			if err = ostream.WriteByte(tag); tag == TagQuote {
				break
			}
		}
	}
	if err == nil {
		var str string
		if str, err = r.readUTF8String(count + 1); err == nil {
			_, err = ostream.WriteString(str)
		}
	}
	return err
}

func (r *RawReader) readGuidRaw(ostream BufWriter) (err error) {
	var guid [38]byte
	if _, err := r.stream.Read(guid[:]); err == nil {
		_, err = ostream.Write(guid[:])
	}
	return err
}

func (r *RawReader) readComplexRaw(ostream BufWriter) (err error) {
	var tag byte
	for tag != TagOpenbrace {
		if tag, err = r.stream.ReadByte(); err == nil {
			err = ostream.WriteByte(tag)
		}
	}
	if err == nil {
		tag, err = r.stream.ReadByte()
	}
	for err == nil && tag != TagClosebrace {
		if err = r.readRaw(ostream, tag); err == nil {
			tag, err = r.stream.ReadByte()
		}
	}
	if err == nil {
		err = ostream.WriteByte(tag)
	}
	return err
}

func (r *RawReader) readUTF8String(length int) (string, error) {
	s := r.stream
	if length == 0 {
		return "", nil
	}
	buf := bytes.NewBuffer(r.strbuf[:0])
	for i := 0; i < length; i++ {
		if r, n, err := s.ReadRune(); err != nil {
			return "", err
		} else {
			buf.WriteRune(r)
			if n > 3 {
				i++
			}
		}
	}
	return string(buf.Bytes()), nil
}

// private functions

func unexpectedTag(tag byte, expectTags []byte) error {
	if t := string([]byte{tag}); expectTags == nil {
		return errors.New("Unexpected serialize tag '" + t + "' in stream")
	} else if bytes.IndexByte(expectTags, tag) < 0 {
		return errors.New("Tag '" + string(expectTags) + "' expected, but '" + t + "' found in stream")
	}
	return nil
}
