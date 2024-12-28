package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

// 用于将数据使用 gob 编解码
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

// 将conn转为Codec
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf), // 创建一个 gob.Encoder，并将缓冲区作为输出目标
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}
// 从输入流读取数据并解码到 body 中
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

// 将header、body发送回去
func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		c.buf.Flush()
		if err != nil {
			c.Close()
		}
	}()
	// 编码header并向缓冲区写入
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	// 编码body并向缓冲区写入
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}
func (c *GobCodec) Close() error {
	return c.conn.Close()
}
