gorets
======

RETS client in Go


The attempt is to meet 1.8.0 compliance.

http://www.reso.org/assets/RETS/Specifications/rets_1_8.pdf.

```
package main


import (
	"flag"
	"fmt"
	"github.com/jpfielding/gorets"
)

func main () {
	username := flag.String("username", "", "Username for the RETS server")
	password := flag.String("password", "", "Password for the RETS server")
	loginUrl := flag.String("login-url", "", "Login URL for the RETS server")
	userAgent := flag.String("user-agent","Threewide/1.5","User agent for the RETS client")

	flag.Parse()

	// should we throw an err here too?
	session, err := gorets.NewSession(*username, *password, *userAgent, "")
	if err != nil {
		panic(err)
	}

	capability, err := session.Login(*loginUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println(capability)

	err = session.Get(capability.Get)
	if err != nil {
		panic(err)
	}

	mUrl := capability.GetMetadata
	format := "COMPACT"
	session.GetMetadata(gorets.MetadataRequest{mUrl, format, "METADATA-SYSTEM", "0"})
	session.GetMetadata(gorets.MetadataRequest{mUrl, format, "METADATA-RESOURCE", "0"})
	session.GetMetadata(gorets.MetadataRequest{mUrl, format, "METADATA-CLASS", "ActiveAgent"})
	session.GetMetadata(gorets.MetadataRequest{mUrl, format, "METADATA-TABLE", "ActiveAgent:ActiveAgent"})

	req := gorets.SearchRequest{
		Url: capability.Search,
		Query: "((LocaleListingStatus=|ACTIVE-CORE),~(VOWList=0))",
		SearchType: "Property",
		Class: "ALL",
		Format: "COMPACT-DECODED",
		QueryType: "DMQL2",
		Count: gorets.COUNT_AFTER,
		Limit: 3,
		Offset: -1,
	}
	result, err := session.Search(req)
	if err != nil {
		panic(err)
	}
	cols := []string{
		"ListingKey",
		"ListPrice",
		"ListingID",
		"TotalPhotos",
		"ModificationTimestamp",
	}
	fmt.Println("COLUMNS:", cols)
	filter := result.FilterTo(cols)
	for row := range result.Data {
		fmt.Println(filter(row))
	}

	session.Logout(capability.Logout)
}
```
