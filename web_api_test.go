package steamweb_test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/leighmacdonald/steamid/v4/steamid"
	"github.com/leighmacdonald/steamweb/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	testIDSquirrelly = steamid.New(76561197961279983) //nolint:gochecknoglobals
	testIDDane       = steamid.New(76561198057999536) //nolint:gochecknoglobals
	testIDMurph      = steamid.New(76561197973805634) //nolint:gochecknoglobals
	testAppTF2       = steamid.AppID(440)             //nolint:gochecknoglobals
)

func TestMain(m *testing.M) {
	if steamweb.Key() == "" {
		panic("steam_api_key unset, SetKey(), or STEAM_TOKEN env key required")
	}

	_ = steamweb.SetLang("en_US")

	os.Exit(m.Run())
}

func TestGetAppList(t *testing.T) {
	apps, err := steamweb.GetAppList(context.Background())
	require.NoError(t, err)
	require.Greater(t, len(apps), 5000)
}

func TestPlayerSummaries(t *testing.T) {
	ids := steamid.Collection{steamid.New(76561198132612090), testIDSquirrelly, steamid.New(76561197960435530)}
	p, err := steamweb.PlayerSummaries(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(p))
}

func TestGetUserGroupList(t *testing.T) {
	groupIDs, err := steamweb.GetUserGroupList(context.Background(), testIDSquirrelly)
	require.NoError(t, err)
	require.Greater(t, len(groupIDs), 5)
}

func TestGetFriendList(t *testing.T) {
	friends, err := steamweb.GetFriendList(context.Background(), steamid.New(76561198132612090))
	require.NoError(t, err)
	require.Greater(t, len(friends), 10)
}

func TestGetPlayerBans(t *testing.T) {
	ids := steamid.Collection{steamid.New(76561198132612090), testIDSquirrelly, steamid.New(76561197960435530)}
	bans, err := steamweb.GetPlayerBans(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(bans))
}

func TestGetServersAtAddress(t *testing.T) {
	servers, err := steamweb.GetServersAtAddress(context.Background(), net.ParseIP("51.222.245.142"))
	require.NoError(t, err)
	require.Positive(t, len(servers))
}

func TestGetServerList(t *testing.T) {
	servers, err := steamweb.GetServerList(context.Background(), map[string]string{"appid": "440"})
	require.NoError(t, err)
	require.Positive(t, len(servers))
}

func TestUpToDateCheck(t *testing.T) {
	respOld, err := steamweb.UpToDateCheck(context.Background(), 440, 100)
	require.NoError(t, err)
	require.True(t, respOld.Success)
	require.False(t, respOld.UpToDate)

	respNew, err2 := steamweb.UpToDateCheck(context.Background(), 440, respOld.RequiredVersion)
	require.NoError(t, err2)
	require.True(t, respNew.Success)
	require.True(t, respNew.UpToDate)
}

func TestGetNewsForApp(t *testing.T) {
	newsItems, err := steamweb.GetNewsForApp(context.Background(), 440, nil)
	require.NoError(t, err)
	require.Greater(t, len(newsItems), 1)

	opts := &steamweb.GetNewsForAppOptions{
		Count: 10,
	}

	newsItemsCount, err2 := steamweb.GetNewsForApp(context.Background(), 440, opts)
	require.NoError(t, err2)
	require.Len(t, newsItemsCount, int(opts.Count))
}

func TestGetNumberOfCurrentPlayers(t *testing.T) {
	players, err := steamweb.GetNumberOfCurrentPlayers(context.Background(), 440)
	require.NoError(t, err)
	require.Greater(t, players, 2000)
}

func TestGetUserStatsForGame(t *testing.T) {
	s, err := steamweb.GetUserStatsForGame(context.Background(), testIDSquirrelly, 440)
	require.NoError(t, err)
	require.Positive(t, len(s.Stats))
	require.Positive(t, len(s.Achievements))
	require.Equal(t, "Team Fortress 2", s.GameName)

	_, err2 := steamweb.GetUserStatsForGame(context.Background(), steamid.New(76561198084134025), 440)
	require.Error(t, err2)
}

func TestGetPlayerItems(t *testing.T) {
	_, backpackSlots, err := steamweb.GetPlayerItems(context.Background(), testIDSquirrelly, 440)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Positive(t, backpackSlots)
}

func TestGetSchemaOverview(t *testing.T) {
	schemaOverview, err := steamweb.GetSchemaOverview(context.Background(), 440)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Positive(t, len(schemaOverview.ItemLevels))
}

func TestGetSchemaItems(t *testing.T) {
	items, err := steamweb.GetSchemaItems(context.Background(), 440)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Greater(t, len(items), 5000)
}

func TestGetSchemaURL(t *testing.T) {
	schemaURL, err := steamweb.GetSchemaURL(context.Background(), 440)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Greater(t, len(schemaURL), 50)
}

func TestGetStoreMetaData(t *testing.T) {
	storeMetaData, err := steamweb.GetStoreMetaData(context.Background(), 440)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Len(t, storeMetaData.PlayerClassData, 9)
}

func TestGetSupportedAPIList(t *testing.T) {
	apiList, err := steamweb.GetSupportedAPIList(context.Background())
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Greater(t, len(apiList), 10)
}

func TestResolveVanityURL(t *testing.T) {
	queries := []string{
		"SQUIRRELLY",
		"  SQUIRRELLY   ",
		"https://steamcommunity.com/id/SQUIRRELLY",
		"https://steamcommunity.com/profiles/76561197961279983",
	}
	for _, s := range queries {
		sid, err := steamweb.ResolveVanityURL(context.Background(), s)
		if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
			t.Skipf("Service not available currently")

			return
		}

		require.NoError(t, err)
		require.Equal(t, testIDSquirrelly, sid)
	}
}

func TestGetSteamLevel(t *testing.T) {
	steamLevel, err := steamweb.GetSteamLevel(context.Background(), testIDSquirrelly)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.GreaterOrEqual(t, steamLevel, 47)
}

func TestGetRecentlyPlayedGames(t *testing.T) {
	recentlyPlayedGames, err := steamweb.GetRecentlyPlayedGames(context.Background(), testIDMurph)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Positive(t, len(recentlyPlayedGames))
}

func TestGetOwnedGames(t *testing.T) {
	ownedGames, err := steamweb.GetOwnedGames(context.Background(), testIDSquirrelly)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.Positive(t, len(ownedGames))
}

func TestGetBadges(t *testing.T) {
	badges, err := steamweb.GetBadges(context.Background(), testIDDane)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.NotNil(t, badges)
	require.Positive(t, len(badges.Badges))
}

func TestGetCommunityBadgeProgress(t *testing.T) {
	badgeProgress, err := steamweb.GetCommunityBadgeProgress(context.Background(), testIDDane)
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)

	// Very flaky test?
	if badgeProgress != nil {
		require.Positive(t, len(badgeProgress))
	}
}

func TestGetAssetClassInfo(t *testing.T) {
	assetClassInfo, err := steamweb.GetAssetClassInfo(context.Background(), testAppTF2, []int{195151, 16891096})
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	require.NoError(t, err)
	require.NotNil(t, assetClassInfo)
}

func TestGetGroupMembers(t *testing.T) {
	groupMembers, err := steamweb.GetGroupMembers(context.TODO(), steamid.New(103582791429521412))
	if err != nil && errors.Is(err, steamweb.ErrServiceUnavailable) {
		t.Skipf("Service not available currently")

		return
	}

	expected := steamid.New(76561197985607672)
	found := false

	for _, sid := range groupMembers {
		if sid.Int64() == expected.Int64() {
			found = true

			break
		}
	}

	require.NoError(t, err)
	require.True(t, found)
}
