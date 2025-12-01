package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/behummble/29-11-2025/internal/models"
	"github.com/jung-kurt/gofpdf"
)

const (
	statusAvaliable = "avaliable"
	statusNotAvaliable = "not avaliable"
)

type LinkService struct {
	log *slog.Logger
	client *http.Client
	storage Storage
	shutdown chan struct{}
}

type siteStatus struct {
	link string
	status string
}

type Storage interface {
	WriteLinksPackage(links []string, ) (int, error)
	Links(packetdID int) (map[string]string, []string, error)
	LinksStatus(links []string) map[string]string
	ValidateCache(newValues map[string]string)
	AllLinks() map[string]string
	UpdateLinksInfo(links map[string]string)
}

func NewService(storage Storage, log *slog.Logger) *LinkService {
	return &LinkService{
		log: log,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		storage: storage,
		shutdown: make(chan struct{}, 1),
	}
}

func(svc *LinkService) Shutdown(ctx context.Context) error {
	select {
	case <- ctx.Done():
		return context.Cause(ctx)
	case svc.shutdown <- struct{}{}:
		return nil
	}
}

func(svc *LinkService) VerifyLinks(ctx context.Context, data []byte) (models.VerifyLinksResponse, error) {
	var linksRequest models.VerifyLinksRequest
	err := json.Unmarshal(data, &linksRequest)
	if err != nil {
		svc.log.Error(
			"ParsingJSONError", 
			slog.String("component", "json/unmarshalling"),
			slog.Any("error", err),
		)
		return models.VerifyLinksResponse{}, errors.New("DecodingDataError")
	}

	if len(linksRequest.Links) == 0 {
		return models.VerifyLinksResponse{}, errors.New("EmptyBody")
	}

	cachedLinks := svc.storage.LinksStatus(linksRequest.Links)
	
	linksInfo := make(map[string]string, len(linksRequest.Links))
	newLinks := make(map[string]string, len(linksRequest.Links) - len(cachedLinks))
	notInCache := make([]string, 0, len(linksInfo))
	for _, link := range linksRequest.Links {
		if _, ok := cachedLinks[link]; !ok {
			notInCache = append(notInCache, link)
		}
	}

	status := make(chan siteStatus, 10)
	svc.linksStatus(status, notInCache)
	
	for siteStatus := range status {
		linksInfo[siteStatus.link] = siteStatus.status
		newLinks[siteStatus.link] = siteStatus.status
	}

	id, err := svc.storage.WriteLinksPackage(linksRequest.Links)
	if err != nil {
		return models.VerifyLinksResponse{}, err
	}

	svc.storage.UpdateLinksInfo(newLinks)

	res := models.VerifyLinksResponse{
		Links: linksInfo,
		Links_num: id,
	}

	return res, nil
}

func(svc *LinkService) PackageLinks(ctx context.Context, data []byte) ([]byte, error) {
	var packageLinksRequest models.LinksPackageRequest
	err := json.Unmarshal(data, &packageLinksRequest)
	if err != nil {
		svc.log.Error(
			"ParsingJSONError", 
			slog.String("component", "json/unmarshalling"),
			slog.Any("error", err),
		)
		return nil, errors.New("DecodingDataError")
	}

	if len(packageLinksRequest.Links_list) == 0 {
		return nil, errors.New("EmptyBody")
	}

	res := make(map[string]string, 1024)
	notInCacheLinks := make(map[string]string, 1024)
	linksToUpdate := make([]string, 0, 1024)
	for _, id := range packageLinksRequest.Links_list {
		links, notInCache, err := svc.storage.Links(id)
		if err != nil {
			svc.log.Error(
				"LinksReadingError", 
				slog.String("component", "storage"),
				slog.Any("error", err),
			)
			return nil, err
		}
		for link, status := range links {
			if _, ok := res[link]; !ok {
				res[link] = status
			}
		}
		linksToUpdate = append(linksToUpdate, notInCache...)
	}

	status := make(chan siteStatus, 10)
	svc.linksStatus(status, linksToUpdate)
	
	for siteStatus := range status {
		res[siteStatus.link] = siteStatus.status
		notInCacheLinks[siteStatus.link] = siteStatus.status
	}

	svc.storage.UpdateLinksInfo(notInCacheLinks)

	return svc.createPDF(res)
}

func(svc *LinkService) ValidateCache() {
	ticker := time.Tick(15 *time.Minute)
	loop:
	for {
		select {
		case <-ticker:
			svc.log.Info("Starting validate cache")
			allLinks := svc.storage.AllLinks()
			newLinks := make(map[string]string, len(allLinks)/6)
			for key, value := range allLinks {
				status := linkStatus(key, svc.log)
				if status != value {
					newLinks[key] = status
				}
			}
			svc.storage.ValidateCache(newLinks)
		case <-svc.shutdown:
			break loop
		}
	}
}

func linkStatus(link string, log *slog.Logger) string {

	resp, err := http.Get(link)
	if err != nil {
		log.Error("Ping site error", slog.String("url", link), slog.String("error", err.Error()))
		return statusNotAvaliable
	}
	
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return statusNotAvaliable
	}
	
	return statusAvaliable
}

func(svc *LinkService) linksStatus(status chan<- siteStatus, links []string) {
	var wg sync.WaitGroup
	wg.Add(len(links))
	for _, link := range links {

		go func(url string) {
			defer wg.Done()
			resp, err := svc.client.Get(fmt.Sprintf("http://%s", link))
			siteStatus := siteStatus{
				link: link,
				status: statusAvaliable,
			}
			if err != nil {
				svc.log.Error("Ping site error", slog.String("url", link), slog.String("error", err.Error()))
				siteStatus.status = statusNotAvaliable
				status<- siteStatus
				return
			}
			
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				siteStatus.status = statusNotAvaliable
			}
			status<- siteStatus

		}(link)
	}
	
	wg.Wait()
	close(status)
}

func(svc *LinkService) createPDF(links map[string]string) ([]byte, error) {
	payload, err := json.Marshal(links)
	if err != nil {
		svc.log.Error(
			"LinksReadingError", 
			slog.String("component", "json/encoding"),
			slog.Any("error", err),
		)
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 10, string(payload))
	buffer := bytes.NewBuffer(payload)
	err = pdf.Output(buffer)
	defer pdf.Close()
	if err != nil {
		svc.log.Error(
			"CreatePDFFileError", 
			slog.String("component", "pdf"),
			slog.Any("error", err),
		)
		return nil, err
	}

	return buffer.Bytes(), nil
}