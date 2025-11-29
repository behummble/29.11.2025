package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/behummble/29-11-2025/internal/models"
)

const (
	statusAvaliable = "avaliable"
	statusNotAvaliable = "not avaliable"
)

type LinkService struct {
	log *slog.Logger
	storage Storage
	shutdown <-chan struct{}
}

type Storage interface {
	WriteLinksPackage(links []string, newLinks map[string]string) (int, error)
	Links(packetdID int) (map[string]string, error)
	LinksStatus(links []string) map[string]string
	ValidateCache(newValues map[string]string)
	AllLinks() map[string]string
}

func NewService(storage Storage, log *slog.Logger, shutdown <-chan struct{}) *LinkService {
	return &LinkService{
		log: log,
		storage: storage,
		shutdown: shutdown,
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
	for _, link := range linksRequest.Links {
		status := cachedLinks[link]
		if _, ok := cachedLinks[link]; !ok {
			status = linkStatus(link, svc.log)
			newLinks[link] = status
		}
		linksInfo[link] = status
	}

	id, err := svc.storage.WriteLinksPackage(linksRequest.Links, newLinks)
	if err != nil {
		return models.VerifyLinksResponse{}, err
	}

	res := models.VerifyLinksResponse{
		Resp: linksInfo,
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

	res := make(map[string]string, 0)

	for _, id := range packageLinksRequest.Links_list {
		links, err := svc.storage.Links(id)
		if err != nil {
			svc.log.Error(
				"LinksReadingError", 
				slog.String("component", "storage"),
				slog.Any("error", err),
			)
			return nil, err
		}
		for k, v := range links {
			if _, ok := res[k]; !ok {
				res[k] = v
			}
		}
	}

	payload, err := json.Marshal(res)
	if err != nil {
		svc.log.Error(
			"LinksReadingError", 
			slog.String("component", "json/encoding"),
			slog.Any("error", err),
		)
		return nil, err
	}
	return payload, err
}

func(svc *LinkService) ValidateCache() {
	ticker := time.Tick(15 *time.Minute)
	for {
		select {
		case <-ticker:
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
			return
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