package database

import (
	"database/sql"
	"encoding/json"

	"github.com/vimcolorschemes/worker/internal/repository"
)

func loadColorschemes(repositoryID int64) ([]repository.Colorscheme, error) {
	rows, err := db.Query(`
		SELECT cs.id, cs.name, csg.background, csg.name, csg.hex_code
		FROM colorschemes cs
		LEFT JOIN colorscheme_groups csg ON csg.colorscheme_id = cs.id
		WHERE cs.repository_id = ?
		ORDER BY cs.id, csg.id`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	type schemeEntry struct {
		index int
	}
	schemeMap := make(map[int64]*schemeEntry)
	var schemes []repository.Colorscheme

	for rows.Next() {
		var schemeID int64
		var schemeName string
		var bg, groupName, hexCode sql.NullString

		if err := rows.Scan(&schemeID, &schemeName, &bg, &groupName, &hexCode); err != nil {
			return nil, err
		}

		entry, exists := schemeMap[schemeID]
		if !exists {
			schemes = append(schemes, repository.Colorscheme{Name: schemeName})
			entry = &schemeEntry{index: len(schemes) - 1}
			schemeMap[schemeID] = entry
		}

		if bg.Valid {
			group := repository.ColorschemeGroup{Name: groupName.String, HexCode: hexCode.String}
			s := &schemes[entry.index]
			switch repository.BackgroundValue(bg.String) {
			case repository.LightBackground:
				s.Data.Light = append(s.Data.Light, group)
			case repository.DarkBackground:
				s.Data.Dark = append(s.Data.Dark, group)
			}
		}
	}

	// Derive backgrounds
	for i := range schemes {
		var backgrounds []repository.BackgroundValue
		if len(schemes[i].Data.Light) > 0 {
			backgrounds = append(backgrounds, repository.LightBackground)
		}
		if len(schemes[i].Data.Dark) > 0 {
			backgrounds = append(backgrounds, repository.DarkBackground)
		}
		schemes[i].Backgrounds = backgrounds
	}

	return schemes, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanRepository(s scannable) (repository.Repository, error) {
	var repo repository.Repository
	var historyJSON string
	var githubCreatedAt, pushedAt, updatedAt sql.NullTime

	err := s.Scan(
		&repo.ID, &repo.Owner.Name, &repo.Owner.AvatarURL, &repo.Name, &repo.GithubURL,
		&repo.StargazersCount, &historyJSON, &repo.WeekStargazersCount,
		&githubCreatedAt, &pushedAt,
		&repo.IsEligible, &updatedAt,
	)
	if err != nil {
		return repository.Repository{}, err
	}

	if githubCreatedAt.Valid {
		repo.GithubCreatedAt = githubCreatedAt.Time
	}
	if pushedAt.Valid {
		repo.PushedAt = pushedAt.Time
	}
	if updatedAt.Valid {
		repo.UpdatedAt = updatedAt.Time
	}
	if err := json.Unmarshal([]byte(historyJSON), &repo.StargazersCountHistory); err != nil {
		return repository.Repository{}, err
	}

	return repo, nil
}

func scanRepositoryWithColorschemes(s scannable) (repository.Repository, error) {
	repo, err := scanRepository(s)
	if err != nil {
		return repository.Repository{}, err
	}

	schemes, err := loadColorschemes(repo.ID)
	if err != nil {
		return repository.Repository{}, err
	}
	repo.Colorschemes = schemes

	return repo, nil
}
