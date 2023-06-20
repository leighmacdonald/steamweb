// Package steamweb provides some basic binding for accessing the steam web api
//
// To properly use this package you must first set a steam api key it can use to authenticate
// with the API. You can obtain a key here https://steamcommunity.com/dev/apikey
//
// A key can be set using steam_webapi.SetKey or using the environment variable STEAM_TOKEN
//
// Some results are cached due to being static content that does not need to be updated frequently. These include:
// GetAppList, GetStoreMetaData, GetSchemaURL, GetSchemaOverview, GetSchemaItems, GetSupportedAPIList
package steamweb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://api.steampowered.com%s?"

var (
	ErrInvalidResponse    = errors.New("Invalid response")
	ErrServiceUnavailable = errors.New("Service Unavailable")
	// ErrNoAPIKey is returned for functions that require an API key to use when one has not been set
	ErrNoAPIKey = errors.New("No steam web api key, to obtain one see: " +
		"https://steamcommunity.com/dev/apikey and call SetKey()")
	apiKey     = ""
	lang       = "en_US"
	cfgMu      = &sync.RWMutex{}
	httpClient = &http.Client{Timeout: time.Second * 10}
)

func init() {
	v, found := os.LookupEnv("STEAM_TOKEN")
	if found && v != "" {
		if err := SetKey(v); err != nil {
			fmt.Printf("Invalid steamid set from env: %v\n", err)
		}
	}
}

// SetKey will set the package level steam webapi key used for requests
//
// You can alternatively set the key with the environment variable `STEAM_TOKEN`
// To get a key see: https://steamcommunity.com/dev/apikey
func SetKey(key string) error {
	if len(key) != 32 && len(key) != 0 {
		return errors.New("Tried to set invalid key, must be 32 chars or 0 to remove it")
	}
	cfgMu.Lock()
	apiKey = key
	cfgMu.Unlock()
	return nil
}

// SetLang sets the package level language to use for results which have translations available
// ISO639-1 language code plus ISO 3166-1 alpha 2 country code of the language to return strings in.
// Some examples include en_US, de_DE, zh_CN, and ko_KR. Default: en_US
//
// The default language used is english (en_US) when no translations exist
func SetLang(newLang string) error {
	if len(newLang) != 5 {
		return errors.New("Invalid ISO_639-1 language code")
	}
	cfgMu.Lock()
	lang = strings.ToLower(newLang)
	cfgMu.Unlock()
	return nil
}

type App struct {
	Appid int    `json:"appid"`
	Name  string `json:"name"`
}

// GetAppList Full list of every publicly facing program in the store/library.
func GetAppList(ctx context.Context) ([]App, error) {
	type response struct {
		Applist struct {
			Apps []App `json:"apps"`
		} `json:"applist"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamApps/GetAppList/v2", nil, &r)
	if err != nil {
		return nil, err
	}
	return r.Applist.Apps, nil
}

// apiRequest is the base function that facilitates all HTTP requests to the API
func apiRequest(ctx context.Context, path string, values url.Values, recv interface{}) error {
	if apiKey == "" {
		return ErrNoAPIKey
	}
	u := fmt.Sprintf(baseURL, path)
	c, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	req, err := http.NewRequestWithContext(c, http.MethodGet, u, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to create new request")
	}
	// TODO Should we make a new instance?
	if values != nil {
		values.Set("key", apiKey)
		values.Set("format", "json")
		req.URL.RawQuery = values.Encode()
	}
	resp, errG := httpClient.Do(req)
	if errG != nil {
		return errors.Wrap(errG, "Failed to perform http request")
	}
	b, errR := io.ReadAll(resp.Body)
	if errR != nil {
		return errors.Wrap(errR, "Failed to read response body")
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusServiceUnavailable {
			return ErrServiceUnavailable
		}
		return errors.Errorf("Invalid status code recieved: %d\n%s", resp.StatusCode, string(b))
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if errU := json.Unmarshal(b, &recv); errU != nil {
		return errors.Wrap(errU, "Failed to decode JSON response")
	}
	return nil
}

// PlayerSummary is the unaltered player summary from the steam official API
type PlayerSummary struct {
	Steamid                  string `json:"steamid"`
	CommunityVisibilityState int    `json:"communityvisibilitystate"`
	ProfileState             int    `json:"profilestate"`
	PersonaName              string `json:"personaname"`
	ProfileURL               string `json:"profileurl"`
	Avatar                   string `json:"avatar"`
	AvatarMedium             string `json:"avatarmedium"`
	AvatarFull               string `json:"avatarfull"`
	AvatarHash               string `json:"avatarhash"`
	PersonaState             int    `json:"personastate"`
	RealName                 string `json:"realname"`
	PrimaryClanID            string `json:"primaryclanid"`
	TimeCreated              int    `json:"timecreated"`
	PersonastateFlags        int    `json:"personastateflags"`
	LocCountryCode           string `json:"loccountrycode"`
	LocStateCode             string `json:"locstatecode"`
	LocCityID                int    `json:"loccityid"`
}

// PlayerSummaries will call GetPlayerSummaries on the valve WebAPI returning the players
// portion of the response as []PlayerSummary
//
// It will only accept up to 100 steamids in a single call
func PlayerSummaries(ctx context.Context, steamIDs steamid.Collection) ([]PlayerSummary, error) {
	type response struct {
		Response struct {
			Players []PlayerSummary `json:"players"`
		} `json:"response"`
	}
	if len(steamIDs) == 0 {
		return nil, errors.New("Too few steam ids, min 1")
	}
	if len(steamIDs) > 100 {
		return nil, errors.New("Too many steam ids, max 100")
	}
	var r response
	err := apiRequest(ctx, "/ISteamUser/GetPlayerSummaries/v0002/", url.Values{
		"steamids": []string{strings.Join(steamIDs.ToStringSlice(), ",")},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Response.Players, err
}

type PlayerBanState struct {
	SteamID          string `json:"SteamId"`
	CommunityBanned  bool   `json:"CommunityBanned"`
	VACBanned        bool   `json:"VACBanned"`
	NumberOfVACBans  int    `json:"NumberOfVACBans"`
	DaysSinceLastBan int    `json:"DaysSinceLastBan"`
	NumberOfGameBans int    `json:"NumberOfGameBans"`
	EconomyBan       string `json:"EconomyBan"`
}

// GetPlayerBans
// https://wiki.teamfortress.com/wiki/WebAPI/GetPlayerBans
func GetPlayerBans(ctx context.Context, steamIDs steamid.Collection) ([]PlayerBanState, error) {
	type response struct {
		Players []PlayerBanState `json:"players"`
	}
	if len(steamIDs) == 0 {
		return nil, errors.New("Too few steam ids, min 1")
	}
	if len(steamIDs) > 100 {
		return nil, errors.New("Too many steam ids, max 100")
	}
	var r response
	err := apiRequest(ctx, "/ISteamUser/GetPlayerBans/v1/", url.Values{
		"steamids": []string{strings.Join(steamIDs.ToStringSlice(), ",")},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Players, err
}

func GetUserGroupList(ctx context.Context, steamID steamid.SID64) ([]steamid.GID, error) {
	type GetUserGroupListResponse struct {
		Response struct {
			Success bool `json:"success"`
			Groups  []struct {
				Gid int64 `json:"gid,string"`
			} `json:"groups"`
		} `json:"response"`
	}
	var r GetUserGroupListResponse
	err := apiRequest(ctx, "/ISteamUser/GetUserGroupList/v1", url.Values{
		"steamid": []string{steamID.String()},
	}, &r)
	if err != nil {
		return nil, err
	}
	var ids []steamid.GID
	for _, v := range r.Response.Groups {
		ids = append(ids, steamid.GID(v.Gid))
	}
	return ids, nil
}

type Friend struct {
	Steamid      string `json:"steamid"`
	Relationship string `json:"relationship"`
	FriendSince  int    `json:"friend_since"`
}

func GetFriendList(ctx context.Context, steamID steamid.SID64) ([]Friend, error) {
	type GetFriendListResponse struct {
		Friendslist struct {
			Friends []Friend `json:"friends"`
		} `json:"friendslist"`
	}
	var r GetFriendListResponse
	err := apiRequest(ctx, "/ISteamUser/GetFriendList/v1", url.Values{
		"steamid": []string{steamID.String()},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Friendslist.Friends, nil
}

type ServerAtAddress struct {
	Addr     string `json:"addr"`
	GmsIndex int    `json:"gmsindex"`
	Appid    int    `json:"appid"`
	Gamedir  string `json:"gamedir"`
	Region   int    `json:"region"`
	Secure   bool   `json:"secure"`
	Lan      bool   `json:"lan"`
	Gameport int    `json:"gameport"`
	Specport int    `json:"specport"`
}

// GetServersAtAddress Shows all steam-compatible servers related to a IPv4 Address.
func GetServersAtAddress(ctx context.Context, ipAddr net.IP) ([]ServerAtAddress, error) {
	type response struct {
		Response struct {
			Success bool              `json:"success"`
			Servers []ServerAtAddress `json:"servers"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamApps/GetServersAtAddress/v0001", url.Values{
		"addr": []string{ipAddr.String()},
	}, &r)
	if err != nil {
		return nil, err
	}
	if !r.Response.Success {
		return nil, errors.New("Invalid response")
	}
	return r.Response.Servers, nil
}

type VersionCheckInfo struct {
	Success           bool   `json:"success"`
	UpToDate          bool   `json:"up_to_date"`
	VersionIsListable bool   `json:"version_is_listable"`
	RequiredVersion   uint32 `json:"required_version"`
	Message           string `json:"message"`
}

// UpToDateCheck Check if a given app version is the most current available.
func UpToDateCheck(ctx context.Context, id steamid.AppID, version uint32) (*VersionCheckInfo, error) {
	type response struct {
		Response VersionCheckInfo `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamApps/UpToDateCheck/v1", url.Values{
		"appid":   []string{fmt.Sprintf("%d", id)},
		"version": []string{fmt.Sprintf("%d", version)},
	}, &r)
	if err != nil {
		return nil, err
	}
	if !r.Response.Success {
		return nil, ErrInvalidResponse
	}
	return &r.Response, nil
}

type GetNewsForAppOptions struct {
	MaxLength uint32   `json:"max_length"`
	EndDate   uint32   `json:"end_date"`
	Count     uint32   `json:"count"`
	Feeds     []string `json:"feeds"`
}

type NewsItem struct {
	Gid           string   `json:"gid"`
	Title         string   `json:"title"`
	URL           string   `json:"url"`
	IsExternalURL bool     `json:"is_external_url"`
	Author        string   `json:"author"`
	Contents      string   `json:"contents"`
	FeedLabel     string   `json:"feedlabel"`
	Date          int      `json:"date"`
	FeedName      string   `json:"feedname"`
	FeedType      int      `json:"feed_type"`
	Appid         int      `json:"appid"`
	Tags          []string `json:"tags,omitempty"`
}

// GetNewsForApp News feed for various games
func GetNewsForApp(ctx context.Context, id steamid.AppID, opts *GetNewsForAppOptions) ([]NewsItem, error) {
	type response struct {
		Appnews struct {
			Appid     int        `json:"appid"`
			NewsItems []NewsItem `json:"newsitems"`
			Count     int        `json:"count"`
		} `json:"appnews"`
	}
	v := url.Values{
		"appid": []string{fmt.Sprintf("%d", id)},
	}
	if opts != nil {
		if opts.MaxLength > 0 {
			v.Set("maxlength", fmt.Sprintf("%d", opts.MaxLength))
		}
		if opts.Count > 0 {
			v.Set("count", fmt.Sprintf("%d", opts.Count))
		}
		if opts.EndDate > 0 {
			v.Set("end_date", fmt.Sprintf("%d", opts.EndDate))
		}
		if len(opts.Feeds) > 0 {
			v.Set("feeds", strings.Join(opts.Feeds, ","))
		}
	}

	var r response
	err := apiRequest(ctx, "/ISteamNews/GetNewsForApp/v0002", v, &r)
	if err != nil {
		return nil, err
	}
	return r.Appnews.NewsItems, nil
}

// GetNumberOfCurrentPlayers Returns the current number of players for an app.
func GetNumberOfCurrentPlayers(ctx context.Context, id steamid.AppID) (int, error) {
	type response struct {
		Response struct {
			PlayerCount int `json:"player_count"`
			Result      int `json:"result"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamUserStats/GetNumberOfCurrentPlayers/v1", url.Values{
		"appid": []string{fmt.Sprintf("%d", id)},
	}, &r)
	if err != nil {
		return 0, err
	}
	if r.Response.Result != 1 {
		return 0, ErrInvalidResponse
	}
	return r.Response.PlayerCount, nil
}

// GetUserStatsForGame currently 500 status with valid requests.
func GetUserStatsForGame(ctx context.Context, steamID steamid.SID64, appID steamid.AppID) (interface{}, error) {
	type response struct {
		Response struct {
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamUserStats/GetUserStatsForGame/v2", url.Values{
		"steamid": []string{steamID.String()},
		"appid":   []string{fmt.Sprintf("%d", appID)},
	}, &r)
	if err != nil {
		return 0, err
	}
	return r.Response, nil
}

type InventoryItem struct {
	ID         int   `json:"id"`
	OriginalID int   `json:"original_id"`
	Defindex   int   `json:"defindex"`
	Level      int   `json:"level"`
	Quality    int   `json:"quality"`
	Inventory  int64 `json:"inventory"`
	Quantity   int   `json:"quantity"`
	Origin     int   `json:"origin"`
	Equipped   []struct {
		Class int `json:"class"`
		Slot  int `json:"slot"`
	} `json:"equipped,omitempty"`
	FlagCannotTrade bool `json:"flag_cannot_trade,omitempty"`
	Attributes      []struct {
		Defindex   int         `json:"defindex"`
		Value      interface{} `json:"value"`
		FloatValue float64     `json:"float_value"`
	} `json:"attributes"`
	FlagCannotCraft bool `json:"flag_cannot_craft,omitempty"`
}

// GetPlayerItems Lists items in a player's backpack.
// https://wiki.teamfortress.com/wiki/WebAPI/GetPlayerItems
func GetPlayerItems(ctx context.Context, steamID steamid.SID64, appID steamid.AppID) ([]InventoryItem, int, error) {
	type response struct {
		Result struct {
			Status           int             `json:"status"`
			NumBackpackSlots int             `json:"num_backpack_slots"`
			Items            []InventoryItem `json:"items"`
		} `json:"result"`
	}
	var r response
	err := apiRequest(ctx, fmt.Sprintf("/IEconItems_%d/GetPlayerItems/v0001/", appID), url.Values{
		"steamid": []string{steamID.String()},
	}, &r)
	if err != nil {
		return nil, 0, err
	}
	return r.Result.Items, r.Result.NumBackpackSlots, nil
}

// GetSchema retain legacy data shape by combining the new GetSchemaOverview and
// GetSchemaItems results.
//func GetSchema(appID steamid.AppID) ([]InventoryItem, error) {
//	return nil, nil
//}

type SchemaOverview struct {
	Status       int    `json:"status"`
	ItemsGameURL string `json:"items_game_url"`
	Qualities    struct {
		Normal         int `json:"Normal"`
		Rarity1        int `json:"rarity1"`
		Rarity2        int `json:"rarity2"`
		Vintage        int `json:"vintage"`
		Rarity3        int `json:"rarity3"`
		Rarity4        int `json:"rarity4"`
		Unique         int `json:"Unique"`
		Community      int `json:"community"`
		Developer      int `json:"developer"`
		SelfMade       int `json:"selfmade"`
		Customized     int `json:"customized"`
		Strange        int `json:"strange"`
		Completed      int `json:"completed"`
		Haunted        int `json:"haunted"`
		Collectors     int `json:"collectors"`
		PaintKitWeapon int `json:"paintkitweapon"`
	} `json:"qualities"`
	OriginNames []struct {
		Origin int    `json:"origin"`
		Name   string `json:"name"`
	} `json:"originNames"`
	Attributes []struct {
		Name              string `json:"name"`
		DefIndex          int    `json:"defindex"`
		AttributeClass    string `json:"attribute_class"`
		DescriptionString string `json:"description_string,omitempty"`
		DescriptionFormat string `json:"description_format,omitempty"`
		EffectType        string `json:"effect_type"`
		Hidden            bool   `json:"hidden"`
		StoredAsInteger   bool   `json:"stored_as_integer"`
	} `json:"attributes"`
	ItemSets []struct {
		ItemSet    string   `json:"item_set"`
		Name       string   `json:"name"`
		Items      []string `json:"items"`
		Attributes []struct {
			Name  string      `json:"name"`
			Class string      `json:"class"`
			Value interface{} `json:"value"`
		} `json:"attributes,omitempty"`
		StoreBundle string `json:"store_bundle,omitempty"`
	} `json:"item_sets"`
	AttributeControlledAttachedParticles []struct {
		System           string `json:"system"`
		ID               int    `json:"id"`
		AttachToRootbone bool   `json:"attach_to_rootbone"`
		Name             string `json:"name"`
		Attachment       string `json:"attachment,omitempty"`
	} `json:"attribute_controlled_attached_particles"`
	ItemLevels []struct {
		Name   string `json:"name"`
		Levels []struct {
			Level         int    `json:"level"`
			RequiredScore int    `json:"required_score"`
			Name          string `json:"name"`
		} `json:"levels"`
	} `json:"item_levels"`
	KillEaterScoreTypes []struct {
		Type      int    `json:"type"`
		TypeName  string `json:"type_name"`
		LevelData string `json:"level_data"`
	} `json:"kill_eater_score_types"`
	StringLookups []struct {
		TableName string `json:"table_name"`
		Strings   []struct {
			Index  int    `json:"index"`
			String string `json:"string"`
		} `json:"strings"`
	} `json:"string_lookups"`
}

// GetSchemaOverview undocumented newer endpoints, replaces GetSchema
// https://github.com/SteamDatabase/SteamTracking/commit/e71a1cd100dc7f35f3f26e94f1bf58e6ce9957c4
func GetSchemaOverview(ctx context.Context, appID steamid.AppID) (*SchemaOverview, error) {
	type response struct {
		Result SchemaOverview `json:"result"`
	}
	var r response
	err := apiRequest(ctx, fmt.Sprintf("/IEconItems_%d/GetSchemaOverview/v0001/", appID), url.Values{}, &r)
	if err != nil {
		return nil, err
	}
	return &r.Result, nil
}

type SchemaItemCapabilities struct {
	Paintable           bool `json:"paintable"`
	Nameable            bool `json:"nameable"`
	CanCraftIfPurchased bool `json:"can_craft_if_purchased"`
	CanGiftWrap         bool `json:"can_gift_wrap"`
	CanCraftCount       bool `json:"can_craft_count"`
	CanCraftMark        bool `json:"can_craft_mark"`
	CanBeRestored       bool `json:"can_be_restored"`
	StrangeParts        bool `json:"strange_parts"`
	CanCardUpgrade      bool `json:"can_card_upgrade"`
	CanStrangify        bool `json:"can_strangify"`
	CanKillstreakify    bool `json:"can_killstreakify"`
	CanConsume          bool `json:"can_consume"`
}

type SchemaItemStyles struct {
	Name string `json:"name"`
}

type SchemaAttributes struct {
	Name  string      `json:"name"`
	Class string      `json:"class"`
	Value interface{} `json:"value"`
}

type SchemaItem struct {
	Name              string                 `json:"name"`
	Defindex          int                    `json:"defindex"`
	ItemClass         string                 `json:"item_class"`
	ItemTypeName      string                 `json:"item_type_name"`
	ItemName          string                 `json:"item_name"`
	ItemDescription   string                 `json:"item_description,omitempty"`
	ProperName        bool                   `json:"proper_name"`
	ItemSlot          string                 `json:"item_slot"`
	ModelPlayer       string                 `json:"model_player"`
	ItemQuality       int                    `json:"item_quality"`
	ImageInventory    string                 `json:"image_inventory"`
	MinIlevel         int                    `json:"min_ilevel"`
	MaxIlevel         int                    `json:"max_ilevel"`
	ImageURL          string                 `json:"image_url"`
	ImageURLLarge     string                 `json:"image_url_large"`
	DropType          string                 `json:"drop_type,omitempty"`
	CraftClass        string                 `json:"craft_class"`
	CraftMaterialType string                 `json:"craft_material_type"`
	Capabilities      SchemaItemCapabilities `json:"capabilities,omitempty"`
	Styles            []SchemaItemStyles     `json:"styles"`
	UsedByClasses     []string               `json:"used_by_classes,omitempty"`
	Attributes        []SchemaAttributes     `json:"attributes,omitempty"`
}

// GetSchemaItems undocumented newer endpoints
// All paged results are fetched and merged
// https://github.com/SteamDatabase/SteamTracking/commit/e71a1cd100dc7f35f3f26e94f1bf58e6ce9957c4
func GetSchemaItems(ctx context.Context, appID steamid.AppID) ([]SchemaItem, error) {
	type response struct {
		Result struct {
			Status       int          `json:"status"`
			ItemsGameURL string       `json:"items_game_url"`
			Items        []SchemaItem `json:"items"`
			Next         int          `json:"next"`
		} `json:"result"`
	}
	var (
		items []SchemaItem
		page  = 0
	)
	for {
		var r response
		err := apiRequest(ctx, fmt.Sprintf("/IEconItems_%d/GetSchemaItems/v1/", appID), url.Values{
			"start": []string{fmt.Sprintf("%d", page)},
		}, &r)
		if err != nil {
			return nil, err
		}
		if r.Result.Next == 0 {
			break
		}
		items = append(items, r.Result.Items...)
		page = r.Result.Next
	}
	return items, nil
}

// GetSchemaURL Returns a URL for the games' item_game.txt file.
func GetSchemaURL(ctx context.Context, appID steamid.AppID) (string, error) {
	type response struct {
		Result struct {
			Status       int    `json:"status"`
			ItemsGameURL string `json:"items_game_url"`
		} `json:"result"`
	}
	var r response
	err := apiRequest(ctx, fmt.Sprintf("/IEconItems_%d/GetSchemaURL/v0001/", appID), url.Values{}, &r)
	if err != nil {
		return "", err
	}
	if r.Result.Status != 1 {
		return "", ErrInvalidResponse
	}
	return r.Result.ItemsGameURL, nil
}

type Banners struct {
	BaseFilename string `json:"basefilename"`
	Action       string `json:"action"`
	Placement    string `json:"placement"`
	ActionParam  string `json:"action_param"`
}

type CarouselData struct {
	MaxDisplayBanners int       `json:"max_display_banners"`
	Banners           []Banners `json:"banners"`
}

type Children struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type Tabs struct {
	Label            string     `json:"label"`
	ID               string     `json:"id"`
	ParentID         int        `json:"parent_id"`
	UseLargeCells    bool       `json:"use_large_cells"`
	Default          bool       `json:"default"`
	Children         []Children `json:"children"`
	Home             bool       `json:"home"`
	DropdownPrefabID int64      `json:"dropdown_prefab_id,omitempty"`
	ParentName       string     `json:"parent_name,omitempty"`
}

type AllElement struct {
	ID            int    `json:"id"`
	LocalizedText string `json:"localized_text"`
}

type Elements struct {
	Name          interface{} `json:"name"`
	LocalizedText string      `json:"localized_text"`
	ID            int         `json:"id"`
}

type Filters struct {
	ID                  int        `json:"id"`
	Name                string     `json:"name"`
	URLHistoryParamName string     `json:"url_history_param_name"`
	AllElement          AllElement `json:"all_element"`
	Elements            []Elements `json:"elements"`
	Count               int        `json:"count"`
}

type Sorters struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	DataType      string `json:"data_type"`
	SortField     string `json:"sort_field"`
	SortReversed  bool   `json:"sort_reversed"`
	LocalizedText string `json:"localized_text"`
}

type SorterIds struct {
	ID int64 `json:"id"`
}

type SortingPrefabs struct {
	ID                  int64       `json:"id"`
	Name                string      `json:"name"`
	URLHistoryParamName string      `json:"url_history_param_name"`
	SorterIds           []SorterIds `json:"sorter_ids"`
}

type Sorting struct {
	Sorters        []Sorters        `json:"sorters"`
	SortingPrefabs []SortingPrefabs `json:"sorting_prefabs"`
}

type Dropdowns struct {
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	LabelText           string `json:"label_text"`
	URLHistoryParamName string `json:"url_history_param_name"`
}

type Config struct {
	DropdownID         int    `json:"dropdown_id"`
	Name               string `json:"name"`
	Enabled            bool   `json:"enabled"`
	DefaultSelectionID int    `json:"default_selection_id"`
}

type Prefabs struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Config []Config `json:"config"`
}

type DropdownData struct {
	Dropdowns []Dropdowns `json:"dropdowns"`
	Prefabs   []Prefabs   `json:"prefabs"`
}

type PlayerClassData struct {
	ID            int    `json:"id"`
	BaseName      string `json:"base_name"`
	LocalizedText string `json:"localized_text"`
}

type PopularItems struct {
	DefIndex int `json:"def_index"`
	Order    int `json:"order"`
}

type HomePageData struct {
	HomeCategoryID int            `json:"home_category_id"`
	PopularItems   []PopularItems `json:"popular_items"`
}

type StoreMetaData struct {
	CarouselData    CarouselData      `json:"carousel_data"`
	Tabs            []Tabs            `json:"tabs"`
	Filters         []Filters         `json:"filters"`
	Sorting         Sorting           `json:"sorting"`
	DropdownData    DropdownData      `json:"dropdown_data"`
	PlayerClassData []PlayerClassData `json:"player_class_data"`
	HomePageData    HomePageData      `json:"home_page_data"`
}

// GetStoreMetaData Returns a URL for the games' item_game.txt file.
func GetStoreMetaData(ctx context.Context, appID steamid.AppID) (*StoreMetaData, error) {
	type response struct {
		Result StoreMetaData `json:"result"`
	}
	var r response
	err := apiRequest(ctx, fmt.Sprintf("/IEconItems_%d/GetStoreMetaData/v0001/", appID), url.Values{}, &r)
	if err != nil {
		return nil, err
	}
	return &r.Result, nil
}

type SupportedAPIMethods struct {
	Name       string                  `json:"name"`
	Version    int                     `json:"version"`
	HttpMethod string                  `json:"httpmethod"`
	Parameters []SupportedAPIParameter `json:"parameters"`
}

type SupportedAPIParameterType string

//goland:noinspection GoUnusedConst
const (
	PTString SupportedAPIParameterType = "string"
	PTUint32 SupportedAPIParameterType = "uint32"
	PTUint64 SupportedAPIParameterType = "uint64"
)

type SupportedAPIParameter struct {
	Name        string                    `json:"name"`
	Type        SupportedAPIParameterType `json:"type"`
	Optional    bool                      `json:"optional"`
	Description string                    `json:"description"`
}

type SupportedAPIInterfaces struct {
	Name    string                `json:"name"`
	Methods []SupportedAPIMethods `json:"methods"`
}

// GetSupportedAPIList Lists all available WebAPI interfaces.
func GetSupportedAPIList(ctx context.Context) ([]SupportedAPIInterfaces, error) {
	type response struct {
		Apilist struct {
			Interfaces []SupportedAPIInterfaces `json:"interfaces"`
		} `json:"apilist"`
	}
	var r response
	err := apiRequest(ctx, "/ISteamWebAPIUtil/GetSupportedAPIList/v0001/", url.Values{}, &r)
	if err != nil {
		return nil, err
	}
	return r.Apilist.Interfaces, nil
}

// ResolveVanityURL Resolve vanity URL parts to a 64 bit ID
func ResolveVanityURL(ctx context.Context, query string) (steamid.SID64, error) {
	type response struct {
		Response struct {
			Steamid steamid.SID64 `json:"steamid"`
			Success int           `json:"success"`
		} `json:"response"`
	}
	const purl = "steamcommunity.com/profiles/"
	query = strings.Replace(query, " ", "", -1)
	if strings.Contains(query, purl) {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}
		output, err := strconv.ParseInt(query[strings.Index(query, purl)+len(purl):], 10, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "Failed to parse int from query")
		}
		if len(strconv.FormatInt(output, 10)) != 17 {
			return 0, errors.Wrapf(err, "Invalid string length")
		}
		return steamid.SID64(output), nil
	} else if strings.Contains(query, "steamcommunity.com/id/") {
		if string(query[len(query)-1]) == "/" {
			query = query[0 : len(query)-1]
		}
		query = query[strings.Index(query, "steamcommunity.com/id/")+len("steamcommunity.com/id/"):]
	}
	var r response
	err := apiRequest(ctx, "/ISteamUser/ResolveVanityURL/v0001/", url.Values{"vanityurl": []string{query}}, &r)
	if err != nil {
		return 0, err
	}
	return r.Response.Steamid, nil
}

// GetSteamLevel Lists all available WebAPI interfaces.
func GetSteamLevel(ctx context.Context, sid steamid.SID64) (int, error) {
	type response struct {
		Response struct {
			// The steam level of the player.
			PlayerLevel int `json:"player_level"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/IPlayerService/GetSteamLevel/v1/", url.Values{
		"steamid": []string{sid.String()},
	}, &r)
	if err != nil {
		return -1, err
	}
	return r.Response.PlayerLevel, nil
}

type RecentGame struct {
	Appid                  steamid.AppID `json:"appid"`
	Name                   string        `json:"name"`
	Playtime2Weeks         int           `json:"playtime_2weeks"`
	PlaytimeForever        int           `json:"playtime_forever"`
	ImgIconURL             string        `json:"img_icon_url"`
	ImgLogoURL             string        `json:"img_logo_url"`
	PlaytimeWindowsForever int           `json:"playtime_windows_forever"`
	PlaytimeMacForever     int           `json:"playtime_mac_forever"`
	PlaytimeLinuxForever   int           `json:"playtime_linux_forever"`
}

// GetRecentlyPlayedGames Lists recently played games
// No results returned is usually due to privacy settings
func GetRecentlyPlayedGames(ctx context.Context, sid steamid.SID64) ([]RecentGame, error) {
	type response struct {
		Response struct {
			TotalCount int          `json:"total_count"`
			Games      []RecentGame `json:"games"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/IPlayerService/GetRecentlyPlayedGames/v1", url.Values{
		"steamid": []string{sid.String()},
		"count":   []string{"10"},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Response.Games, nil
}

type OwnedGame struct {
	// An integer containing the program's ID.
	Appid steamid.AppID `json:"appid"`
	// A string containing the program's publicly facing title.
	Name string `json:"name"`
	// An integer of the the player's total playtime, denoted in minutes.
	PlaytimeForever int `json:"playtime_forever"`
	// The program icon's file name see: IconURL
	ImgIconURL string `json:"img_icon_url"`
	// The program logo's file name see: LogoURL
	ImgLogoURL               string `json:"img_logo_url"`
	PlaytimeWindowsForever   int    `json:"playtime_windows_forever"`
	PlaytimeMacForever       int    `json:"playtime_mac_forever"`
	PlaytimeLinuxForever     int    `json:"playtime_linux_forever"`
	HasCommunityVisibleStats bool   `json:"has_community_visible_stats,omitempty"`
	// An integer of the player's playtime in the past 2 weeks, denoted in minutes.
	Playtime2Weeks int `json:"playtime_2weeks,omitempty"`
}

func (g OwnedGame) IconURL() string {
	return fmt.Sprintf("https://media.steampowered.com/steamcommunity/public/images/apps/%d/%s.jpg", g.Appid, g.ImgIconURL)
}

func (g OwnedGame) LogoURL() string {
	return fmt.Sprintf("https://media.steampowered.com/steamcommunity/public/images/apps/%d/%s.jpg", g.Appid, g.ImgLogoURL)
}

// GetOwnedGames Lists all owned games
// No results returned is usually due to privacy settings
func GetOwnedGames(ctx context.Context, sid steamid.SID64) ([]OwnedGame, error) {
	type response struct {
		Response struct {
			GameCount int         `json:"game_count"`
			Games     []OwnedGame `json:"games"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/IPlayerService/GetOwnedGames/v1", url.Values{
		"steamid":                   []string{sid.String()},
		"include_appinfo":           []string{"true"},
		"include_played_free_games": []string{"true"},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Response.Games, nil
}

type Badge struct {
	// BadgeID. Currently no official badge schema is available.
	BadgeId int `json:"badgeid"`
	Level   int `json:"level"`
	// Unix timestamp of when the steam user acquired the badge.
	CompletionTime int `json:"completion_time"`
	// The experience this badge is worth, contributing toward the steam account's player_xp.
	Xp int `json:"xp"`
	// The amount of people who has this badge.
	Scarcity int `json:"scarcity"`
	// Provided if the badge relates to an app (trading cards).
	Appid steamid.AppID `json:"appid,omitempty"`
	// Provided if the badge relates to an app (trading cards); the value doesn't seem to be an item
	// in the steam accounts backpack, however the value minus 1 seems to be the item ID for the
	// emoticon granted for crafting this badge, and the value minus 2 seems to be the background granted.
	CommunityItemId string `json:"communityitemid,omitempty"`
	// Provided if the badge relates to an app (trading cards).
	BorderColor int `json:"border_color,omitempty"`
}

type BadgeStatus struct {
	Badges                     []Badge `json:"badges"`
	PlayerXp                   int     `json:"player_xp"`
	PlayerLevel                int     `json:"player_level"`
	PlayerXpNeededToLevelUp    int     `json:"player_xp_needed_to_level_up"`
	PlayerXpNeededCurrentLevel int     `json:"player_xp_needed_current_level"`
}

// GetBadges Lists all badges for a user
// No results returned is usually due to privacy settings
func GetBadges(ctx context.Context, sid steamid.SID64) (*BadgeStatus, error) {
	type response struct {
		Response BadgeStatus `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/IPlayerService/GetBadges/v1", url.Values{
		"steamid": []string{sid.String()},
	}, &r)
	if err != nil {
		return nil, err
	}
	return &r.Response, nil
}

type BadgeQuestStatus struct {
	// Quest ID; no schema is currently available.
	QuestId int `json:"questid"`
	// Whether the steam account has completed this quest.
	Completed bool `json:"completed"`
}

// GetCommunityBadgeProgress Lists all badges for a user
// No results returned is usually due to privacy settings
func GetCommunityBadgeProgress(ctx context.Context, sid steamid.SID64) ([]BadgeQuestStatus, error) {
	type response struct {
		Response struct {
			// Array of quests (actions required to unlock a badge)
			Quests []BadgeQuestStatus `json:"quests"`
		} `json:"response"`
	}
	var r response
	err := apiRequest(ctx, "/IPlayerService/GetCommunityBadgeProgress/v1", url.Values{
		"steamid": []string{sid.String()},
	}, &r)
	if err != nil {
		return nil, err
	}
	return r.Response.Quests, nil
}

type Asset struct {
	//Descriptions []struct {
	//	Name  string `json:"name" mapstructure:"name"`
	//	Value string `json:"value" mapstructure:"value"`
	//	Color string `json:"color,omitempty" mapstructure:"color"`
	//} `json:"descriptions" mapstructure:"descriptions,omitempty"`
	Descriptions    interface{} `json:"descriptions" mapstructure:"descriptions"`
	Fraudwarnings   interface{} `json:"fraudwarnings" mapstructure:"fraudwarnings"`
	Tradable        string      `json:"tradable" mapstructure:"tradable"`
	BackgroundColor string      `json:"background_color" mapstructure:"background_color"`
	IconURL         string      `json:"icon_url" mapstructure:"icon_url"`
	Name            string      `json:"name" mapstructure:"name"`
	Type            string      `json:"type" mapstructure:"type"`
	NameColor       string      `json:"name_color" mapstructure:"name_color"`
	Actions         interface{} `json:"actions" mapstructure:"actions"`
}

// GetAssetClassInfo gets info on items/assets
func GetAssetClassInfo(ctx context.Context, appID steamid.AppID, classIds []int) ([]Asset, error) {
	type response struct {
		Result map[string]interface{} `json:"result"`
	}
	v := url.Values{
		"appid": []string{fmt.Sprintf("%d", appID)},
		// The ISO639-1 language code for the language all localized strings should be returned in.
		// Not all strings have been translated to every language. If a language does not have a string,
		// the English string will be returned instead. If this parameter is omitted the string token will
		// be returned for the strings.
		"language":    []string{lang},
		"class_count": []string{fmt.Sprintf("%d", len(classIds))},
	}
	for i := 0; i < len(classIds); i++ {
		//v.Set(fmt.Sprintf("class_name%d", i), "x")
		v.Set(fmt.Sprintf("classid%d", i), fmt.Sprintf("%d", classIds[i]))
	}
	var r response
	err := apiRequest(ctx, "/ISteamEconomy/GetAssetClassInfo/v0001", v, &r)
	if err != nil {
		return nil, err
	}
	success, found := r.Result["success"]
	if !found || !success.(bool) {
		return nil, ErrInvalidResponse
	}
	delete(r.Result, "success")
	var assets []Asset
	for _, val := range r.Result {
		var s Asset
		if errD := mapstructure.Decode(val, &s); errD != nil {
			return nil, errD
		}
		assets = append(assets, s)
	}
	return assets, nil
}

func GetGroupMembers(ctx context.Context, groupId steamid.GID) (steamid.Collection, error) {
	rx := regexp.MustCompile(`<steamID64>(\d+)</steamID64>`)
	if !groupId.Valid() {
		return nil, errors.New("Invalid steam group ID")
	}
	lCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()
	req, reqErr := http.NewRequestWithContext(lCtx, "GET", fmt.Sprintf("https://steamcommunity.com/gid/%d/memberslistxml/?xml=1", groupId), nil)
	if reqErr != nil {
		return nil, errors.Wrapf(reqErr, "Failed to create request")
	}
	resp, respErr := httpClient.Do(req)
	if respErr != nil {
		return nil, errors.Wrapf(reqErr, "Failed to perform request")
	}
	body, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, errors.Wrapf(reqErr, "Failed to read response body")
	}
	var found steamid.Collection
	for _, match := range rx.FindAllStringSubmatch(string(body), -1) {
		sid, errSid := steamid.StringToSID64(match[1])
		if errSid != nil {
			return nil, errors.Wrapf(errSid, "Found invalid ID: %s", match[1])
		}
		found = append(found, sid)
	}
	return found, nil
}
