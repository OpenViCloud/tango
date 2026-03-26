package rest

import (
	"errors"
	"strconv"

	"tango/internal/application/command"
	"tango/internal/application/query"
	"tango/internal/contract/common"
	"tango/internal/domain"
	response "tango/internal/handler/rest/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	createUser     *command.CreateUserHandler
	updateUser     *command.UpdateUserHandler
	banUser        *command.BanUserHandler
	deleteUser     *command.DeleteUserHandler
	assignUserRole *command.AssignUserRoleHandler
	removeUserRole *command.RemoveUserRoleHandler
	getUserByID    *query.GetUserByIDHandler
	listUsers      *query.ListUsersHandler
	listUserRoles  *query.ListUserRolesHandler
}

func NewUserHandler(
	createUser *command.CreateUserHandler,
	updateUser *command.UpdateUserHandler,
	banUser *command.BanUserHandler,
	deleteUser *command.DeleteUserHandler,
	assignUserRole *command.AssignUserRoleHandler,
	removeUserRole *command.RemoveUserRoleHandler,
	getUserByID *query.GetUserByIDHandler,
	listUsers *query.ListUsersHandler,
	listUserRoles *query.ListUserRolesHandler,
) *UserHandler {
	return &UserHandler{
		createUser:     createUser,
		updateUser:     updateUser,
		banUser:        banUser,
		deleteUser:     deleteUser,
		assignUserRole: assignUserRole,
		removeUserRole: removeUserRole,
		getUserByID:    getUserByID,
		listUsers:      listUsers,
		listUserRoles:  listUserRoles,
	}
}

type createUserRequest struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Password  string `json:"password"`
}

type updateUserRequest struct {
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
}

type assignUserRoleRequest struct {
	RoleID string `json:"role_id"`
}

type userRoleResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsSystem    bool   `json:"is_system"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *UserHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/user/me", h.GetMe)
	rg.GET("/user/:id", h.GetByID)
	rg.GET("/users", h.List)
	rg.POST("/users", h.Create)
	rg.PUT("/users/:id", h.Update)
	rg.POST("/users/:id/ban", h.Ban)
	rg.DELETE("/users/:id", h.Delete)
	rg.GET("/users/:id/roles", h.ListRoles)
	rg.POST("/users/:id/roles", h.AssignRole)
	rg.DELETE("/users/:id/roles/:roleId", h.RemoveRole)
}

// GetMe godoc
// @Summary Get current user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} userResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /user/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	h.getUser(c, c.GetString("user_id"))
}

// GetByID godoc
// @Summary Get user by ID
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} userResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /user/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	h.getUser(c, c.Param("id"))
}

func (h *UserHandler) getUser(c *gin.Context, id string) {
	user, err := h.getUserByID.Handle(c.Request.Context(), query.GetUserByIDQuery{ID: id})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toResponse(user))
}

// List godoc
// @Summary List users
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {array} userResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users [get]
func (h *UserHandler) List(c *gin.Context) {
	req := common.BaseRequestModel{
		PageIndex:  parseIntDefault(c.Query("pageIndex"), 0),
		PageSize:   parseIntDefault(c.Query("pageSize"), 20),
		SearchText: c.Query("searchText"),
		OrderBy:    c.Query("orderBy"),
		Ascending:  c.Query("ascending") == "true",
	}
	result, err := h.listUsers.Handle(c.Request.Context(), query.ListUsersQuery{BaseRequestModel: req})
	if err != nil {
		writeError(c, err)
		return
	}
	resp := make([]userResponse, len(result.Items))
	for i, user := range result.Items {
		resp[i] = toResponse(user)
	}
	totalPage := 0
	if req.PageSize > 0 {
		totalPage = int((result.TotalItems + int64(req.PageSize) - 1) / int64(req.PageSize))
	}
	response.OK(c, common.BaseResponseModel[userResponse]{
		Items:      resp,
		PageIndex:  req.PageIndex,
		PageSize:   req.PageSize,
		TotalItems: result.TotalItems,
		TotalPage:  totalPage,
	})
}

// Create godoc
// @Summary Create user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body createUserRequest true "User payload"
// @Success 201 {object} userResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	user, err := h.createUser.Handle(c.Request.Context(), command.CreateUserCommand{
		ID:        req.ID,
		Email:     req.Email,
		Nickname:  req.Nickname,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Address:   req.Address,
		Password:  req.Password,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Created(c, toResponse(user))
}

// Update godoc
// @Summary Update user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body updateUserRequest true "Updated profile fields"
// @Success 200 {object} userResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	user, err := h.updateUser.Handle(c.Request.Context(), command.UpdateUserCommand{
		ID:        c.Param("id"),
		Nickname:  req.Nickname,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Phone:     req.Phone,
		Address:   req.Address,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.OK(c, toResponse(user))
}

// Ban godoc
// @Summary Ban user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id}/ban [post]
func (h *UserHandler) Ban(c *gin.Context) {
	if err := h.banUser.Handle(c.Request.Context(), command.BanUserCommand{ID: c.Param("id")}); err != nil {
		writeError(c, err)
		return
	}
	response.NoContent(c)
}

// Delete godoc
// @Summary Delete user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 204
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.deleteUser.Handle(c.Request.Context(), command.DeleteUserCommand{ID: c.Param("id")}); err != nil {
		writeError(c, err)
		return
	}
	response.NoContent(c)
}

// ListRoles godoc
// @Summary List roles assigned to a user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {array} userRoleResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id}/roles [get]
func (h *UserHandler) ListRoles(c *gin.Context) {
	roles, err := h.listUserRoles.Handle(c.Request.Context(), query.ListUserRolesQuery{UserID: c.Param("id")})
	if err != nil {
		writeError(c, err)
		return
	}
	resp := make([]userRoleResponse, len(roles))
	for i, role := range roles {
		resp[i] = toRoleResponse(role)
	}
	response.OK(c, resp)
}

// AssignRole godoc
// @Summary Assign role to user
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param request body assignUserRoleRequest true "Role assignment payload"
// @Success 204
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id}/roles [post]
func (h *UserHandler) AssignRole(c *gin.Context) {
	var req assignUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(response.Validation(nil, err.Error()))
		return
	}
	if err := h.assignUserRole.Handle(c.Request.Context(), command.AssignUserRoleCommand{
		UserID: c.Param("id"),
		RoleID: req.RoleID,
	}); err != nil {
		writeError(c, err)
		return
	}
	response.NoContent(c)
}

// RemoveRole godoc
// @Summary Remove role from user
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param roleId path string true "Role ID"
// @Success 204
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /users/{id}/roles/{roleId} [delete]
func (h *UserHandler) RemoveRole(c *gin.Context) {
	if err := h.removeUserRole.Handle(c.Request.Context(), command.RemoveUserRoleCommand{
		UserID: c.Param("id"),
		RoleID: c.Param("roleId"),
	}); err != nil {
		writeError(c, err)
		return
	}
	response.NoContent(c)
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Address   string `json:"address"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func toResponse(user *domain.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Nickname:  user.Nickname,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
		Address:   user.Address,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt.Format(timeLayout),
	}
}

func toRoleResponse(role *domain.Role) userRoleResponse {
	return userRoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		CreatedAt:   role.CreatedAt.Format(timeLayout),
		UpdatedAt:   role.UpdatedAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		_ = c.Error(response.NotFound(err.Error()))
	case errors.Is(err, domain.ErrUserAlreadyExists):
		_ = c.Error(response.Conflict(err.Error()))
	case errors.Is(err, domain.ErrInvalidInput):
		_ = c.Error(response.BadRequest(err.Error()))
	default:
		_ = c.Error(response.InternalCause(err, ""))
	}
}

func parseIntDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
