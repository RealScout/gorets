/**
provides the photo extraction core testing
*/
package rets

import (
	"io/ioutil"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	testutils "github.com/jpfielding/gotest/testutils"
)

func TestGetObject(t *testing.T) {
	header := http.Header{}
	textproto.MIMEHeader(header).Add("Content-Type", "image/jpeg")
	textproto.MIMEHeader(header).Add("Content-ID", "123456")
	textproto.MIMEHeader(header).Add("Object-ID", "1")
	textproto.MIMEHeader(header).Add("Preferred", "1")
	textproto.MIMEHeader(header).Add("UID", "1a234234234")
	textproto.MIMEHeader(header).Add("Content-Description", "Outhouse")
	textproto.MIMEHeader(header).Add("Content-Sub-Description", "The urinal")
	textproto.MIMEHeader(header).Add("Location", "http://www.simpleboundary.com/image-5.jpg")

	var body = `<binary data 1>`

	response := GetObjectResponse{
		Response: &http.Response{
			Header: header,
			Body:   ioutil.NopCloser(strings.NewReader(body)),
		},
	}
	defer response.Close()
	var objects []*Object
	err := response.ForEach(func(o *Object, err error) error {
		objects = append(objects, o)
		return nil
	})
	testutils.Ok(t, err)

	testutils.Equals(t, 1, len(objects))

	o := objects[0]
	testutils.Equals(t, true, o.Preferred)
	testutils.Equals(t, "image/jpeg", o.ContentType)
	testutils.Equals(t, "123456", o.ContentID)
	testutils.Equals(t, 1, o.ObjectID)
	testutils.Equals(t, "1a234234234", o.UID)
	testutils.Equals(t, "Outhouse", o.Description)
	testutils.Equals(t, "The urinal", o.SubDescription)
	testutils.Equals(t, "<binary data 1>", string(o.Blob))
	testutils.Equals(t, "http://www.simpleboundary.com/image-5.jpg", o.Location)
	testutils.Equals(t, false, o.RetsError)
}

var boundary = "simple boundary"

var contentType = `multipart/parallel; boundary="simple boundary"`

var multipartBody = `--simple boundary
Content-Type: image/jpeg
Content-ID: 123456
Object-ID: 1
Preferred: 1
ObjectData: ListingKey=123456
ObjectData: ListDate=2013-05-01T12:34:34.8-0500

<binary data 1>
--simple boundary
Content-Type: image/jpeg
Content-ID: 123456
Object-ID: 2
UID: 1a234234234

<binary data 2>
--simple boundary
Content-Type: image/jpeg
Content-ID: 123456
Object-ID: 3
Content-Description: Outhouse
Content-Sub-Description: The urinal

<binary data 3>
--simple boundary
Content-Type: text/xml
Content-ID: 123457
Object-ID: 4
RETS-Error: 1

<RETS ReplyCode="20403" ReplyText="There is no object with that Object-ID"/>

--simple boundary
Content-Type: image/jpeg
Content-ID: 123457
Object-ID: 5
Location: http://www.simpleboundary.com/image-5.jpg


--simple boundary
Content-Type: image/jpeg
Content-ID: 123457
Object-ID: 6
Location: http://www.simpleboundary.com/image-6.jpg

<binary data 6>
--simple boundary
Content-Type: image/jpeg
Content-ID: 123457
Object-ID: 7
Location: http://www.simpleboundary.com/image-7.jpg

<RETS ReplyCode="0" ReplyText="Found it!"/>
--simple boundary--`

func TestGetObjects(t *testing.T) {
	headers := http.Header{}
	headers.Add("Content-Type", contentType)
	response := GetObjectResponse{
		Response: &http.Response{
			Header: headers,
			Body:   ioutil.NopCloser(strings.NewReader(multipartBody)),
		},
	}
	defer response.Close()
	var objects []*Object
	response.ForEach(func(o *Object, err error) error {
		testutils.Ok(t, err)
		objects = append(objects, o)
		return nil
	})

	o1 := objects[0]
	testutils.Equals(t, true, o1.Preferred)
	testutils.Equals(t, "image/jpeg", o1.ContentType)
	testutils.Equals(t, "123456", o1.ContentID)
	testutils.Equals(t, 1, o1.ObjectID)
	testutils.Equals(t, "<binary data 1>", string(o1.Blob))
	testutils.Equals(t, "123456", o1.ObjectData["ListingKey"])
	testutils.Equals(t, "2013-05-01T12:34:34.8-0500", o1.ObjectData["ListDate"])

	o2 := objects[1]
	testutils.Equals(t, 2, o2.ObjectID)
	testutils.Equals(t, "1a234234234", o2.UID)

	o3 := objects[2]
	testutils.Equals(t, 3, o3.ObjectID)
	testutils.Equals(t, "Outhouse", o3.Description)
	testutils.Equals(t, "The urinal", o3.SubDescription)

	o4 := objects[3]
	testutils.Equals(t, true, o4.RetsError)

	testutils.Equals(t, "text/xml", o4.ContentType)
	testutils.Equals(t, "There is no object with that Object-ID", o4.RetsMessage.Text)
	testutils.Equals(t, StatusObjectNotFound, o4.RetsMessage.Code)

	o5 := objects[4]
	testutils.Equals(t, "http://www.simpleboundary.com/image-5.jpg", o5.Location)
	testutils.Equals(t, "image/jpeg", o5.ContentType)
	testutils.Equals(t, "123457", o5.ContentID)
	testutils.Equals(t, 5, o5.ObjectID)
	testutils.Equals(t, "", string(o5.Blob))

	o6 := objects[5]
	testutils.Equals(t, "http://www.simpleboundary.com/image-6.jpg", o6.Location)
	testutils.Equals(t, "image/jpeg", o6.ContentType)
	testutils.Equals(t, "123457", o6.ContentID)
	testutils.Equals(t, 6, o6.ObjectID)
	testutils.Equals(t, "<binary data 6>", string(o6.Blob))
	testutils.Assert(t, o6.RetsMessage == nil, "should not be the zerod object")

	o7 := objects[6]
	testutils.Equals(t, "http://www.simpleboundary.com/image-7.jpg", o7.Location)
	testutils.Equals(t, "image/jpeg", o7.ContentType)
	testutils.Equals(t, "123457", o7.ContentID)
	testutils.Equals(t, 7, o7.ObjectID)
	testutils.Equals(t, "", string(o7.Blob))
	testutils.Equals(t, "Found it!", o7.RetsMessage.Text)

}
