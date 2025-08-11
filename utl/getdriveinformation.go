// Package utl (getdriveinformation.go) :
// These method is for retrieving Drive Information.
package utl

import (
	"net/url"
	"path"
)

// getDriveInf : Get drive information using Drive API.
func (p *FileInf) getDriveInf() error {
	u, err := url.Parse(driveapiv3)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, "about")
	q := u.Query()
	q.Set("fields", p.SearchFields)
	u.RawQuery = q.Encode()
	r := &RequestParams{
		Method:      "GET",
		APIURL:      u.String(),
		Data:        nil,
		Accesstoken: p.Accesstoken,
		Dtime:       30,
	}
	p.reqAndGetRawResponse(r)
	return nil
}

// GetDriveInformation : Get Drive Information.
// doGetDriveInformation retrieves Drive Information.
func doGetDriveInformation(fileInf *FileInf) *FileInf {
	// TODO: Refactor internal logic to use passed arguments instead of FileInf fields
	// For now, returning the passed FileInf to allow compilation.
	return fileInf
}
