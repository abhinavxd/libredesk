package user

import (
	"database/sql"
	"strings"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
)

func (u *Manager) GetAllSkills() ([]models.Skill, error) {
	out := make([]models.Skill, 0)
	if err := u.q.GetAllSkills.Select(&out); err != nil {
		u.lo.Error("error listing skills", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func (u *Manager) CreateSkill(name string) (models.Skill, error) {
	var skill models.Skill
	name = strings.TrimSpace(name)
	if err := u.q.InsertSkill.Get(&skill, name); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return skill, envelope.NewError(envelope.ConflictError, u.i18n.T("errors.alreadyExistsTag"), nil)
		}
		u.lo.Error("error creating skill", "error", err)
		return skill, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return skill, nil
}

func (u *Manager) UpdateSkill(id int, name string) (models.Skill, error) {
	var skill models.Skill
	if err := u.q.UpdateSkill.Get(&skill, id, strings.TrimSpace(name)); err != nil {
		u.lo.Error("error updating skill", "error", err)
		return skill, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return skill, nil
}

func (u *Manager) DeleteSkill(id int) error {
	if _, err := u.q.DeleteSkill.Exec(id); err != nil {
		u.lo.Error("error deleting skill", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (u *Manager) ListAgentSkills(userID int) ([]models.Skill, error) {
	out := make([]models.Skill, 0)
	if err := u.q.ListAgentSkills.Select(&out, userID); err != nil {
		u.lo.Error("error listing agent skills", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return out, nil
}

func (u *Manager) SetAgentSkills(userID int, skillIDs []int) error {
	if skillIDs == nil {
		skillIDs = []int{}
	}
	if _, err := u.q.SetAgentSkills.Exec(userID, pq.Array(skillIDs)); err != nil {
		u.lo.Error("error setting agent skills", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

func (u *Manager) PickAgentBySkill(skillID int, teamID int) (int, error) {
	var id int
	if err := u.q.PickAgentBySkill.Get(&id, skillID, teamID); err != nil {
		if err == sql.ErrNoRows {
			return 0, envelope.NewError(envelope.InputError, u.i18n.T("skill.noEligibleAgent"), nil)
		}
		u.lo.Error("error picking agent by skill", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return id, nil
}
