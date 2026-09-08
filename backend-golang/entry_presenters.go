package main

import domainentries "kpi-police/backend-golang/internal/domain/entries"

func listedEntriesOut(entries []domainentries.ListedEntry) []EntryOut {
	result := make([]EntryOut, 0, len(entries))
	for _, entry := range entries {
		result = append(result, listedEntryOut(entry))
	}
	return result
}

func listedEntryOut(entry domainentries.ListedEntry) EntryOut {
	attachments := make([]AttachmentOut, 0, len(entry.Attachments))
	for _, attachment := range entry.Attachments {
		attachments = append(attachments, AttachmentOut{
			ID: attachment.ID, Name: attachment.Name,
			ContentType: attachment.ContentType, Size: attachment.Size,
		})
	}
	return EntryOut{
		ID:                    entry.ID,
		Date:                  entry.Date,
		Month:                 entry.Month,
		OfficerID:             entry.OfficerID,
		OfficerName:           entry.OfficerName,
		Rank:                  entry.Rank,
		DistrictName:          entry.DistrictName,
		Code:                  entry.Code,
		Indicator:             entry.Indicator,
		Block:                 entry.Block,
		Subblock:              entry.Subblock,
		NPA:                   entry.NPA,
		SourceURL:             entry.SourceURL,
		Quantity:              round2(entry.Quantity),
		CountedQuantity:       round2(entry.CountedQuantity),
		MonthlyLimit:          round2(entry.MonthlyLimit),
		IsMass:                entry.IsMass,
		IsCritical:            entry.IsCritical,
		PointsPerUnit:         round2(entry.PointsPerUnit),
		Coeff:                 round2(entry.Coeff),
		RawScore:              round2(entry.RawScore),
		CappedScore:           round2(entry.CappedScore),
		AntiCoeff:             round2(entry.AntiCoeff),
		FinalScore:            round2(entry.FinalScore),
		Status:                entry.Status,
		Comments:              entry.Comments,
		HeadComment:           entry.HeadComment,
		AttachmentName:        entry.AttachmentName,
		AttachmentContentType: entry.AttachmentContentType,
		AttachmentSize:        entry.AttachmentSize,
		Attachments:           attachments,
	}
}
