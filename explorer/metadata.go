package explorer

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/jpfielding/gorets/metadata"
	"github.com/jpfielding/gorets/rets"
	"github.com/jpfielding/gorets/retsutil"
)

// MetadataService ...
type MetadataService struct{}

// MetadataGetParams ...
type MetadataGetParams struct {
	ID         string `json:"id"`
	Extraction string // (|STANDARD-XML|COMPACT|COMPACT-INCREMENTAL) the format to pull from the server
}

// MetadataGetResponse ...
type MetadataGetResponse struct {
	Metadata metadata.MSystem
}

// Get ....
func (ms MetadataService) Get(r *http.Request, args *MetadataGetParams, reply *MetadataGetResponse) error {
	fmt.Printf("params: %v\n", args)

	c := ConnectionService{}.Load()[args.ID]
	if JSONExist(c.MSystem()) {
		standard := metadata.MSystem{}
		JSONLoad(c.MSystem(), &standard)
		reply.Metadata = standard
		return nil
	}
	op, ok := options[args.Extraction]
	if !ok {
		return fmt.Errorf("%s not supported", args.Extraction)
	}
	if args.Extraction == "" {
		args.Extraction = "COMPACT"
	}
	// lookup the operation for pulling metadata
	ctx := context.Background()
	rqr, err := c.Login(ctx)
	if err != nil {
		return err
	}
	standard, err := op(rqr, ctx, c.URLs.GetMetadata)
	if err != nil {
		return err
	}
	reply.Metadata = *standard
	JSONStore(c.MSystem(), &standard)
	return nil
}

// Metadata ...
// input: MetadataParams
// output: metadata.MSystem
func Metadata() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var p MetadataGetParams
		if r.Body != nil {
			json.NewDecoder(r.Body).Decode(&p)
		}
		fmt.Printf("params: %v\n", p)

		c := ConnectionService{}.Load()[p.ID]
		if JSONExist(c.MSystem()) {
			standard := metadata.MSystem{}
			JSONLoad(c.MSystem(), &standard)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&standard)
			return
		}

		if op, ok := options[p.Extraction]; ok {
			if p.Extraction == "" {
				p.Extraction = "COMPACT"
			}
			// lookup the operation for pulling metadata
			ctx := context.Background()
			r, err := c.Login(ctx)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			standard, err := op(r, ctx, c.URLs.GetMetadata)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(standard)
			JSONStore(c.MSystem(), &standard)
		} else {
			http.Error(w, fmt.Sprintf("%s not supported", p.Extraction), 400)
			return
		}
	}
}

// MetadataRequestType is a typedef metadata extraction options
type MetadataRequestType func(requester rets.Requester, ctx context.Context, url string) (*metadata.MSystem, error)

// options for extracting metadata
var options = map[string]MetadataRequestType{
	"STANDARD-XML":        getStandardMetadata,
	"COMPACT":             getCompactMetadata,
	"COMPACT-INCREMENTAL": getCompactIncremental,
}

// TODO extract a common func and switch on the incoming param

// getCompactIncremental retrieve the RETS Compact metadata from the server
func getCompactIncremental(requester rets.Requester, ctx context.Context, url string) (*metadata.MSystem, error) {
	compact := &retsutil.IncrementalCompact{}
	err := compact.Load(requester, ctx, url)
	if err != nil {
		return nil, err
	}
	return retsutil.AsStandard(*compact).Convert()
}

// getCompactMetadata retrieve the RETS Compact metadata from the server
func getCompactMetadata(requester rets.Requester, ctx context.Context, url string) (*metadata.MSystem, error) {
	reader, err := rets.MetadataStream(requester, ctx, rets.MetadataRequest{
		URL:    url,
		Format: "COMPACT",
		MType:  "METADATA-SYSTEM",
		ID:     "*",
	})
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	compact, err := rets.ParseMetadataCompactResult(reader)
	if err != nil {
		return nil, err
	}
	return retsutil.AsStandard(*compact).Convert()
}

// getStandardMetadata ...
func getStandardMetadata(requester rets.Requester, ctx context.Context, url string) (*metadata.MSystem, error) {
	reader, err := rets.MetadataStream(requester, ctx, rets.MetadataRequest{
		URL:    url,
		Format: "STANDARD-XML",
		MType:  "METADATA-SYSTEM",
		ID:     "*",
	})
	defer reader.Close()
	if err != nil {
		return nil, err
	}
	parser := xml.NewDecoder(reader)
	rets := metadata.RETSResponseWrapper{}
	err = parser.Decode(&rets)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return &rets.Metadata.MSystem, err
}