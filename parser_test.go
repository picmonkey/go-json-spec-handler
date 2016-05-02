package jsh

import (
	"encoding/json"
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestParsing(t *testing.T) {

	Convey("Parse Tests", t, func() {

		Convey("->validateHeaders()", func() {
			Convey("should reject invalid Content-Type", func() {
				req, reqErr := http.NewRequest("GET", "", nil)
				So(reqErr, ShouldBeNil)
				req.Header.Set("Content-Type", "jpeg")

				err := validateHeaders(req.Header)
				So(err, ShouldNotBeNil)
				So(err.Status, ShouldEqual, http.StatusNotAcceptable)
			})

			Convey("should work with valid Content-Type", func() {
				req, reqErr := http.NewRequest("GET", "", nil)
				So(reqErr, ShouldBeNil)
				req.Header.Set("Content-Type", "application/vnd.api+json")

				err := validateHeaders(req.Header)
				So(err, ShouldBeNil)
			})

			Convey("should tolerate correct Content-Type with parameters to work around Firefox < v.43 bug (BREAKS SPEC: http://jsonapi.org/format/#content-negotiation-clients)", func() {
				req, reqErr := http.NewRequest("GET", "", nil)
				So(reqErr, ShouldBeNil)
				req.Header.Set("Content-Type", "application/vnd.api+json; charset=UTF-8")

				err := validateHeaders(req.Header)
				So(err, ShouldBeNil)
			})
		})

		Convey("->ParseObject()", func() {

			Convey("should parse a valid object", func() {

				objectJSON := `{
					"data": {
						"type": "user",
						"id": "sweetID123",
						"attributes": {"ID":"123"},
						"relationships": {
							"company": {
								"data": { "type": "company", "id": "companyID123" }
							},
							"comments": {
								"data": [
									{ "type": "comments", "id": "commentID123" },
									{ "type": "comments", "id": "commentID456" }
								]
							}
						}
					}
				}`
				req, reqErr := testRequest([]byte(objectJSON))
				So(reqErr, ShouldBeNil)

				object, err := ParseObject(req)

				So(err, ShouldBeNil)
				So(object, ShouldNotBeEmpty)
				So(object.Type, ShouldEqual, "user")
				So(object.ID, ShouldEqual, "sweetID123")
				So(object.Attributes, ShouldResemble, json.RawMessage(`{"ID":"123"}`))
				So(object.Relationships["company"], ShouldResemble, &Relationship{Data: ResourceLinkage{&ResourceIdentifier{Type: "company", ID: "companyID123"}}})
				So(object.Relationships["comments"], ShouldResemble, &Relationship{Data: ResourceLinkage{{Type: "comments", ID: "commentID123"}, {Type: "comments", ID: "commentID456"}}})
			})

			Convey("should reject an object with missing attributes", func() {
				objectJSON := `{"data": {"id": "sweetID123", "attributes": {"ID":"123"}}}`

				req, reqErr := testRequest([]byte(objectJSON))
				So(reqErr, ShouldBeNil)

				_, err := ParseObject(req)
				So(err, ShouldNotBeNil)
				So(err.Status, ShouldEqual, 422)
				So(err.Source.Pointer, ShouldEqual, "/data/attributes/type")
			})

			Convey("should accept empty ID only for POST", func() {
				objectJSON := `{"data": {"id": "", "type":"test", "attributes": {"ID":"123"}}}`
				req, reqErr := testRequest([]byte(objectJSON))
				So(reqErr, ShouldBeNil)

				Convey("POST test", func() {
					req.Method = "POST"
					_, err := ParseObject(req)
					So(err, ShouldBeNil)
				})

				Convey("PATCH test", func() {
					req.Method = "PATCH"
					_, err := ParseObject(req)
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("->ParseList()", func() {

			Convey("should parse a valid list", func() {

				listJSON :=
					`{"data": [
		{"type": "user", "id": "sweetID123", "attributes": {"ID":"123"}},
		{"type": "user", "id": "sweetID456", "attributes": {"ID":"456"}}
		]}`
				req, reqErr := testRequest([]byte(listJSON))
				So(reqErr, ShouldBeNil)

				list, err := ParseList(req)
				So(err, ShouldBeNil)
				So(len(list), ShouldEqual, 2)

				object := list[1]
				So(object.Type, ShouldEqual, "user")
				So(object.ID, ShouldEqual, "sweetID456")
				So(object.Attributes, ShouldResemble, json.RawMessage(`{"ID":"456"}`))
			})

			Convey("should error for an invalid list", func() {
				listJSON :=
					`{"data": [
		{"type": "user", "id": "sweetID123", "attributes": {"ID":"123"}},
		{"type": "user", "attributes": {"ID":"456"}}
		]}`

				req, reqErr := testRequest([]byte(listJSON))
				So(reqErr, ShouldBeNil)

				_, err := ParseList(req)
				So(err, ShouldNotBeNil)
				So(err.Status, ShouldEqual, 422)
				So(err.Source.Pointer, ShouldEqual, "/data/attributes/id")
			})
		})
	})
}
