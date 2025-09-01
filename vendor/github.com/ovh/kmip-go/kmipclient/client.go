package kmipclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"

	"github.com/ovh/kmip-go"
	"github.com/ovh/kmip-go/payloads"
	"github.com/ovh/kmip-go/ttlv"
)

var supportedVersions = []kmip.ProtocolVersion{kmip.V1_4, kmip.V1_3, kmip.V1_2, kmip.V1_1, kmip.V1_0}

type opts struct {
	middlewares       []Middleware
	supportedVersions []kmip.ProtocolVersion
	enforceVersion    *kmip.ProtocolVersion
	rootCAs           [][]byte
	certs             []tls.Certificate
	serverName        string
	tlsCfg            *tls.Config
	tlsCiphers        []uint16
	//TODO: Add KMIP Authentication / Credentials
	//TODO: Overwrite default/preferred/supported key formats for register
}

func (o *opts) tlsConfig() (*tls.Config, error) {
	cfg := o.tlsCfg
	if cfg == nil {
		cfg = &tls.Config{
			MinVersion: tls.VersionTLS12, // As required by KMIP 1.4 spec

			// CipherSuites: []uint16{
			// 	// Mandatory support as per KMIP 1.4 spec
			// 	// tls.TLS_RSA_WITH_AES_256_CBC_SHA256, // Not supported in Go
			// 	tls.TLS_RSA_WITH_AES_128_CBC_SHA256, // insecure

			// 	// Optional support as per KMIP 1.4 spec
			// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			// 	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			// 	tls.TLS_RSA_WITH_AES_128_CBC_SHA,            // insecure
			// 	tls.TLS_RSA_WITH_AES_256_CBC_SHA,            // insecure
			// 	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, // insecure
			// 	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,      // insecure
			// 	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,   // insecure
			// },
		}
	}
	if cfg.RootCAs == nil {
		if len(o.rootCAs) > 0 {
			cfg.RootCAs = x509.NewCertPool()
		} else {
			var err error
			if cfg.RootCAs, err = x509.SystemCertPool(); err != nil {
				return nil, err
			}
		}
	}
	for _, ca := range o.rootCAs {
		cfg.RootCAs.AppendCertsFromPEM(ca)
	}
	cfg.Certificates = append(cfg.Certificates, o.certs...)
	if cfg.ServerName == "" {
		cfg.ServerName = o.serverName
	}

	for _, cipher := range o.tlsCiphers {
		if !slices.Contains(cfg.CipherSuites, cipher) {
			cfg.CipherSuites = append(cfg.CipherSuites, cipher)
		}
	}

	return cfg, nil
}

type Option func(*opts) error

func WithMiddlewares(middlewares ...Middleware) Option {
	return func(o *opts) error {
		o.middlewares = append(o.middlewares, middlewares...)
		return nil
	}
}

func WithKmipVersions(versions ...kmip.ProtocolVersion) Option {
	return func(o *opts) error {
		o.supportedVersions = append(o.supportedVersions, versions...)
		slices.SortFunc(o.supportedVersions, func(a, b kmip.ProtocolVersion) int {
			return ttlv.CompareVersions(b, a)
		})
		o.supportedVersions = slices.Compact(o.supportedVersions)
		return nil
	}
}

func EnforceVersion(v kmip.ProtocolVersion) Option {
	return func(o *opts) error {
		o.enforceVersion = &v
		return nil
	}
}

// WithRootCAFile adds the CA in the file located at `path` t othe clients
// CA pool. If path is an empty string, the option is a no-op.
func WithRootCAFile(path string) Option {
	return func(o *opts) error {
		if path == "" {
			return nil
		}
		pem, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		o.rootCAs = append(o.rootCAs, pem)
		return nil
	}
}

func WithRootCAPem(pem []byte) Option {
	return func(o *opts) error {
		o.rootCAs = append(o.rootCAs, pem)
		return nil
	}
}

func WithClientCert(cert tls.Certificate) Option {
	return func(o *opts) error {
		o.certs = append(o.certs, cert)
		return nil
	}
}

func WithClientCertFiles(certFile, keyFile string) Option {
	return func(o *opts) error {
		tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
		o.certs = append(o.certs, tlsCert)
		return nil
	}
}

func WithClientCertPEM(certPEMBlock, keyPEMBlock []byte) Option {
	return func(o *opts) error {
		tlsCert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return err
		}
		o.certs = append(o.certs, tlsCert)
		return nil
	}
}

func WithServerName(name string) Option {
	return func(o *opts) error {
		o.serverName = name
		return nil
	}
}

func WithTlsConfig(cfg *tls.Config) Option {
	return func(o *opts) error {
		o.tlsCfg = cfg
		return nil
	}
}

func WithTlsCipherSuiteNames(ciphers ...string) Option {
	return func(o *opts) error {
	search:
		for _, cipherName := range ciphers {
			for _, s := range tls.CipherSuites() {
				if s.Name != cipherName {
					continue
				}
				o.tlsCiphers = append(o.tlsCiphers, s.ID)
				continue search
			}
			for _, s := range tls.InsecureCipherSuites() {
				if s.Name != cipherName {
					continue
				}
				o.tlsCiphers = append(o.tlsCiphers, s.ID)
				continue search
			}
			return fmt.Errorf("invalid TLS cipher name %q", cipherName)
		}
		return nil
	}
}

func WithTlsCipherSuites(ciphers ...uint16) Option {
	return func(o *opts) error {
		o.tlsCiphers = append(o.tlsCiphers, ciphers...)
		return nil
	}
}

type Client struct {
	lock              *sync.Mutex
	conn              *conn
	version           *kmip.ProtocolVersion
	supportedVersions []kmip.ProtocolVersion
	dialer            func(context.Context) (*conn, error)
	middlewares       []Middleware
	addr              string
}

func Dial(addr string, options ...Option) (*Client, error) {
	return DialContext(context.Background(), addr, options...)
}

func DialContext(ctx context.Context, addr string, options ...Option) (*Client, error) {
	opts := opts{}
	for _, o := range options {
		if err := o(&opts); err != nil {
			return nil, err
		}
	}
	if len(opts.supportedVersions) == 0 {
		opts.supportedVersions = append(opts.supportedVersions, supportedVersions...)
	}

	tlsCfg, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}

	dialer := func(ctx context.Context) (*conn, error) {
		tlsDialer := tls.Dialer{
			Config: tlsCfg.Clone(),
		}
		conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		return newConn(conn), nil
	}

	stream, err := dialer(ctx)
	if err != nil {
		return nil, err
	}

	c := &Client{
		lock:              new(sync.Mutex),
		conn:              stream,
		dialer:            dialer,
		supportedVersions: opts.supportedVersions,
		version:           opts.enforceVersion,
		middlewares:       opts.middlewares,
		addr:              addr,
	}

	// Negotiate protocol version
	if err := c.negotiateVersion(ctx); err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

// Clone is like CloneCtx but uses internally a background context.
func (c *Client) Clone() (*Client, error) {
	return c.CloneCtx(context.Background())
}

// CloneCtx clones the current kmip client into a new independent client
// with a separate new connection. The new client inherits allt he configured parameters
// as well as the negotiated kmip protocol version. Meaning that cloning a client does not perform
// protocol version negotiation.
//
// Cloning a closed client is valid and will create a new connected client.
func (c *Client) CloneCtx(ctx context.Context) (*Client, error) {
	stream, err := c.dialer(ctx)
	if err != nil {
		return nil, err
	}
	version := *c.version
	return &Client{
		lock:              new(sync.Mutex),
		version:           &version,
		supportedVersions: slices.Clone(c.supportedVersions),
		dialer:            c.dialer,
		middlewares:       slices.Clone(c.middlewares),
		conn:              stream,
		addr:              c.addr,
	}, nil
}

func (c *Client) Version() kmip.ProtocolVersion {
	return *c.version
}

func (c *Client) Addr() string {
	return c.addr
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) reconnect(ctx context.Context) error {
	// fmt.Println("Reconnecting")
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	stream, err := c.dialer(ctx)
	if err != nil {
		return err
	}
	c.conn = stream
	return nil
}

func (c *Client) doRountrip(ctx context.Context, msg *kmip.RequestMessage) (*kmip.ResponseMessage, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.conn == nil {
		if err := c.reconnect(ctx); err != nil {
			return nil, err
		}
	}

	//TODO: Better reconnection loop. Do we really need a retry counter here ?
	retry := 3
	for {
		resp, err := c.conn.roundtrip(ctx, msg)
		if err == nil {
			return resp, nil
		}
		if retry <= 0 || (!errors.Is(err, io.EOF) && !errors.Is(err, io.ErrClosedPipe)) {
			return nil, err
		}
		if err := c.reconnect(ctx); err != nil {
			return nil, err
		}
		retry--
	}
}

func (c *Client) Roundtrip(ctx context.Context, msg *kmip.RequestMessage) (*kmip.ResponseMessage, error) {
	i := 0
	var next func(ctx context.Context, req *kmip.RequestMessage) (*kmip.ResponseMessage, error)
	next = func(ctx context.Context, req *kmip.RequestMessage) (*kmip.ResponseMessage, error) {
		if i < len(c.middlewares) {
			mdl := c.middlewares[i]
			i++
			return mdl(next, ctx, req)
		}
		return c.doRountrip(ctx, req)
	}
	return next(ctx, msg)
}

func (c *Client) negotiateVersion(ctx context.Context) error {
	if c.version != nil {
		return nil
	}
	msg := kmip.NewRequestMessage(kmip.V1_1, &payloads.DiscoverVersionsRequestPayload{
		ProtocolVersion: c.supportedVersions,
	})

	resp, err := c.Roundtrip(ctx, &msg)
	if err != nil {
		return err
	}
	if resp.Header.BatchCount != 1 || len(resp.BatchItem) != 1 {
		return errors.New("Unexpected batch item count")
	}
	bi := resp.BatchItem[0]
	if bi.ResultStatus == kmip.ResultStatusOperationFailed && bi.ResultReason == kmip.ResultReasonOperationNotSupported {
		// If the discover opertion is not supported, then fallbacks to kmip v1.0
		// but also check that v1.0 is in the client's supported version list and return an error if not.
		if !slices.Contains(c.supportedVersions, kmip.V1_0) {
			return errors.New("Protocol version negotiation failed. No common version found")
		}
		c.version = &kmip.V1_0
		return nil
	}
	if err := bi.Err(); err != nil {
		return err
	}
	serverVersions := bi.ResponsePayload.(*payloads.DiscoverVersionsResponsePayload).ProtocolVersion
	if len(serverVersions) == 0 {
		return errors.New("Protocol version negotiation failed. No common version found")
	}
	c.version = &serverVersions[0]
	return nil
}

func (c *Client) Request(ctx context.Context, payload kmip.OperationPayload) (kmip.OperationPayload, error) {
	resp, err := c.Batch(ctx, payload)
	if err != nil {
		return nil, err
	}
	bi := resp[0]
	if err := bi.Err(); err != nil {
		return nil, err
	}
	return bi.ResponsePayload, nil
}

func (c *Client) Batch(ctx context.Context, payloads ...kmip.OperationPayload) ([]kmip.ResponseBatchItem, error) {
	msg := kmip.NewRequestMessage(*c.version, payloads...)
	resp, err := c.Roundtrip(ctx, &msg)
	if err != nil {
		return nil, err
	}
	// Check batch item count
	if int(resp.Header.BatchCount) != len(resp.BatchItem) || len(resp.BatchItem) != len(payloads) {
		return nil, errors.New("Batch count mismatch")
	}
	return resp.BatchItem, nil
}

type Executor[Req, Resp kmip.OperationPayload] struct {
	client interface {
		Request(context.Context, kmip.OperationPayload) (kmip.OperationPayload, error)
		Version() kmip.ProtocolVersion
	}
	req Req
	err error
}

// Exec sends the request to the remote KMIP server, and returns the parsed response.
//
// It returns an error if the request could not be sent, or if the server replies with
// KMIP error.
func (ex Executor[Req, Resp]) Exec() (Resp, error) {
	return ex.ExecContext(context.Background())
}

// ExecContext sends the request to the remote KMIP server, and returns the parsed response.
//
// It returns an error if the request could not be sent, or if the server replies with
// KMIP error.
func (ex Executor[Req, Resp]) ExecContext(ctx context.Context) (Resp, error) {
	if ex.err != nil {
		var zero Resp
		return zero, fmt.Errorf("Request initialization failed: %w", ex.err)
	}
	resp, err := ex.client.Request(ctx, ex.req)
	if err != nil {
		var zero Resp
		return zero, err
	}
	return resp.(Resp), nil
}

// MustExec is like Exec except it panics if the request fails.
func (ex Executor[Req, Resp]) MustExec() Resp {
	return ex.MustExecContext(context.Background())
}

// MustExecContext is like Exec except it panics if the request fails.
func (ex Executor[Req, Resp]) MustExecContext(ctx context.Context) Resp {
	resp, err := ex.ExecContext(ctx)
	if err != nil {
		//TODO: Add operation ID string
		panic(fmt.Errorf("Request failed: %w", err))
	}
	return resp
}

func (ex Executor[Req, Resp]) RequestPayload() Req {
	return ex.req
}

type AttributeExecutor[Req, Resp kmip.OperationPayload, Wrap any] struct {
	Executor[Req, Resp]
	attrFunc func(*Req) *[]kmip.Attribute
	wrap     func(AttributeExecutor[Req, Resp, Wrap]) Wrap
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithAttributes(attributes ...kmip.Attribute) Wrap {
	attrPtr := ex.attrFunc(&ex.req)
	*attrPtr = append(*attrPtr, attributes...)
	return ex.wrap(ex)
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithAttribute(name kmip.AttributeName, value any) Wrap {
	return ex.WithAttributes(kmip.Attribute{AttributeName: name, AttributeIndex: nil, AttributeValue: value})
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithUniqueID(id string) Wrap {
	return ex.WithAttribute(kmip.AttributeNameUniqueIdentifier, id)
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithName(name string) Wrap {
	return ex.WithAttribute(kmip.AttributeNameName, kmip.Name{
		NameValue: name,
		NameType:  kmip.NameTypeUninterpretedTextString,
	})
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithURI(uri string) Wrap {
	return ex.WithAttribute(kmip.AttributeNameName, kmip.Name{
		NameValue: uri,
		NameType:  kmip.NameTypeUri,
	})
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithLink(linkType kmip.LinkType, linkedObjectID string) Wrap {
	return ex.WithAttribute(kmip.AttributeNameLink, kmip.Link{
		LinkType:               linkType,
		LinkedObjectIdentifier: linkedObjectID,
	})
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithObjectType(objectType kmip.ObjectType) Wrap {
	//TODO: Ignore zero value
	return ex.WithAttribute(kmip.AttributeNameObjectType, objectType)
}

func (ex AttributeExecutor[Req, Resp, Wrap]) WithUsageLimit(total int64, unit kmip.UsageLimitsUnit) Wrap {
	return ex.WithAttribute(kmip.AttributeNameUsageLimits, kmip.UsageLimits{
		UsageLimitsTotal: total,
		UsageLimitsCount: &total,
		UsageLimitsUnit:  unit,
	})
}
