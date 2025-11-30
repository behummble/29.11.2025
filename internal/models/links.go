package models

type VerifyLinksRequest struct {
	Links []string
}

type VerifyLinksResponse struct {
	Links map[string]string
	Links_num int
}

type LinksPackageRequest struct {
	Links_list []int
}