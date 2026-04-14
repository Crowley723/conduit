package jobs

import (
	"context"
	"errors"
	"fmt"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/models"
	"github.com/Crowley723/conduit/internal/utils"
	"time"
)

type CertificateIssuedStatusJob struct {
	appCtx   *middlewares.AppContext
	interval time.Duration
}

func NewCertificateIssuedStatusJob(appCtx *middlewares.AppContext, interval time.Duration) *CertificateIssuedStatusJob {
	return &CertificateIssuedStatusJob{
		appCtx:   appCtx,
		interval: interval,
	}
}

func (j *CertificateIssuedStatusJob) Name() string {
	return "certificate_issued_status"
}

func (j *CertificateIssuedStatusJob) RequiresLeadership() bool {
	return true
}

func (j *CertificateIssuedStatusJob) Interval() time.Duration {
	return j.interval
}

func (j *CertificateIssuedStatusJob) Run(ctx context.Context) error {
	if j.interval <= 0 {
		j.appCtx.Logger.Error("certificate issued status job failed: ticker interval must not be zero")
		return fmt.Errorf("non-positive ticker interval: %s", j.interval)
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	certs, err := getIssuedCertificates(j.appCtx)
	if err != nil {
		if !errors.Is(err, errNoIssuedCertificates) {
			j.appCtx.Logger.Error("error checking for issued certificate", "error", err)
		}
	} else {
		err = handleIssuedCertificates(j.appCtx, certs)
		if err != nil {
			j.appCtx.Logger.Error("unable to update issued certificate status", "error", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			j.appCtx.Logger.Debug("Background issued certificate job canceled")
			return ctx.Err()
		case <-ticker.C:
			certs, err := getIssuedCertificates(j.appCtx)
			if err != nil {
				if !errors.Is(err, errNoIssuedCertificates) {
					j.appCtx.Logger.Error("unable to check for issued certificate", "error", err)
				}
				continue
			}

			err = handleIssuedCertificates(j.appCtx, certs)
			if err != nil {
				j.appCtx.Logger.Error("unable to update issued certificate status", "error", err)
			}
		}
	}
}

func getIssuedCertificates(ctx *middlewares.AppContext) ([]*models.CertificateRequest, error) {
	certs, err := ctx.Storage.GetPendingCertificateRequests(ctx)
	if err != nil {
		return nil, err
	}

	if len(certs) == 0 || certs == nil {
		return nil, errNoIssuedCertificates
	}

	return certs, nil
}

func handleIssuedCertificates(ctx *middlewares.AppContext, certs []*models.CertificateRequest) error {
	systemIss, systemSub, err := ctx.Storage.GetSystemUser(ctx)
	if err != nil {
		return fmt.Errorf("error getting system user: %w", err)
	}

	for _, cert := range certs {
		if cert.CertificateIdentifier == nil {
			ctx.Logger.Error("certificate request missing identifier", "request_id", cert.ID)
			continue
		}

		// Check if certificate is ready
		ready, err := ctx.CertificateManager.IsCertificateReady(ctx, *cert.CertificateIdentifier)
		if err != nil {
			ctx.Logger.Error("error checking certificate readiness", "error", err, "identifier", *cert.CertificateIdentifier)
			continue
		}

		if !ready {
			continue
		}

		certPEM, _, _, err := ctx.CertificateManager.GetCertificateData(ctx, *cert.CertificateIdentifier)
		if err != nil {
			ctx.Logger.Error("unable to get certificate PEM", "error", err, "identifier", *cert.CertificateIdentifier)
			continue
		}

		certDetails, err := utils.ParseCertificateDetails(certPEM)
		if err != nil {
			ctx.Logger.Error("unable to parse certificate PEM", "error", err, "identifier", *cert.CertificateIdentifier)
			continue
		}

		err = ctx.Storage.UpdateCertificateRequestIssued(ctx, cert.ID, string(certPEM), certDetails.SerialNumber, certDetails.NotBefore, certDetails.NotAfter, systemIss, systemSub)
		if err != nil {
			ctx.Logger.Error("unable to update certificate status", "error", err, "request_id", cert.ID)
			continue
		}

		ctx.Logger.Debug("Certificate Issuance Completed",
			"request_id", cert.ID,
			"identifier", *cert.CertificateIdentifier)
	}

	return nil
}
