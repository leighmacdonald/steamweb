package steamweb

import (
	"context"
	"fmt"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net"
	"os"
	"testing"
)

const (
	testIDSquirrelly = steamid.SID64(76561197961279983)
	testIDDane       = steamid.SID64(76561198057999536)
	testIDMurph      = steamid.SID64(76561197973805634)
	testAppTF2       = steamid.AppID(440)
)

func TestMain(m *testing.M) {
	if apiKey == "" {
		fmt.Println("steam_api_key unset, SetKey(), or STEAM_TOKEN env key required")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestGetAppList(t *testing.T) {
	apps, err := GetAppList(context.Background())
	require.NoError(t, err)
	require.True(t, len(apps) > 5000)
}

func TestPlayerSummaries(t *testing.T) {
	ids := steamid.Collection{76561198132612090, testIDSquirrelly, 76561197960435530}
	p, err := PlayerSummaries(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(p))
}

func TestGetUserGroupList(t *testing.T) {
	groupIDs, err := GetUserGroupList(context.Background(), testIDSquirrelly)
	require.NoError(t, err)
	require.True(t, len(groupIDs) > 5)
}

func TestGetFriendList(t *testing.T) {
	friends, err := GetFriendList(context.Background(), 76561198132612090)
	require.NoError(t, err)
	require.True(t, len(friends) > 10)
}

func TestGetPlayerBans(t *testing.T) {
	ids := steamid.Collection{76561198132612090, testIDSquirrelly, 76561197960435530}
	bans, err := GetPlayerBans(context.Background(), ids)
	require.NoError(t, err)
	require.Equal(t, len(ids), len(bans))
}

func TestGetServersAtAddress(t *testing.T) {
	servers, err := GetServersAtAddress(context.Background(), net.ParseIP("51.222.245.142"))
	require.NoError(t, err)
	require.True(t, len(servers) > 0)
}

func TestUpToDateCheck(t *testing.T) {
	respOld, err := UpToDateCheck(context.Background(), 440, 100)
	require.NoError(t, err)
	require.True(t, respOld.Success)
	require.False(t, respOld.UpToDate)
	respNew, err2 := UpToDateCheck(context.Background(), 440, respOld.RequiredVersion)
	require.NoError(t, err2)
	require.True(t, respNew.Success)
	require.True(t, respNew.UpToDate)
}

func TestGetNewsForApp(t *testing.T) {
	newsItems, err := GetNewsForApp(context.Background(), 440, nil)
	require.NoError(t, err)
	require.True(t, len(newsItems) > 1)
	opts := &GetNewsForAppOptions{
		Count: 10,
	}
	newsItemsCount, err := GetNewsForApp(context.Background(), 440, opts)
	require.NoError(t, err)
	require.Equal(t, int(opts.Count), len(newsItemsCount))
}

func TestGetNumberOfCurrentPlayers(t *testing.T) {
	players, err := GetNumberOfCurrentPlayers(context.Background(), 440)
	require.NoError(t, err)
	require.Greater(t, players, 2000)
}

func TestGetUserStatsForGame(t *testing.T) {
	t.Skipf("Service not currently functional")
	return
	_, err := GetUserStatsForGame(context.Background(), testIDSquirrelly, 440)
	require.Error(t, err)

	_, err2 := GetUserStatsForGame(context.Background(), 76561198084134025, 440)
	require.NoError(t, err2)
}

func TestGetPlayerItems(t *testing.T) {
	_, c, err := GetPlayerItems(context.Background(), testIDSquirrelly, 440)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, c > 0)
}

func TestGetSchemaOverview(t *testing.T) {
	s, err := GetSchemaOverview(context.Background(), 440)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, len(s.ItemLevels) > 0)
}

func TestGetSchemaItems(t *testing.T) {
	items, err := GetSchemaItems(context.Background(), 440)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.Greater(t, len(items), 5000)
}

func TestGetSchemaURL(t *testing.T) {
	schemaUrl, err := GetSchemaURL(context.Background(), 440)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, len(schemaUrl) > 50)
}

func TestGetStoreMetaData(t *testing.T) {
	md, err := GetStoreMetaData(context.Background(), 440)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.Equal(t, 9, len(md.PlayerClassData))
}

func TestGetSupportedAPIList(t *testing.T) {
	apiList, err := GetSupportedAPIList(context.Background())
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, len(apiList) > 10)
}

func TestResolveVanityURL(t *testing.T) {
	queries := []string{
		"SQUIRRELLY",
		"  SQUIRRELLY   ",
		"https://steamcommunity.com/id/SQUIRRELLY",
		"https://steamcommunity.com/profiles/76561197961279983"}
	for _, s := range queries {
		sid, err := ResolveVanityURL(context.Background(), s)
		if err != nil && errors.Is(err, ErrServiceUnavailable) {
			t.Skipf("Service not available currently")
			return
		}
		require.NoError(t, err)
		require.Equal(t, testIDSquirrelly, sid)
	}
}

func TestGetSteamLevel(t *testing.T) {
	sl, err := GetSteamLevel(context.Background(), testIDSquirrelly)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, sl >= 47)
}

func TestGetRecentlyPlayedGames(t *testing.T) {
	rp, err := GetRecentlyPlayedGames(context.Background(), testIDMurph)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, len(rp) > 0)
}

func TestGetOwnedGames(t *testing.T) {
	rp, err := GetOwnedGames(context.Background(), testIDDane)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.True(t, len(rp) > 0)
}

func TestGetBadges(t *testing.T) {
	bds, err := GetBadges(context.Background(), testIDDane)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.NotNil(t, bds)
	require.True(t, len(bds.Badges) > 0)
}

func TestGetCommunityBadgeProgress(t *testing.T) {
	bds, err := GetCommunityBadgeProgress(context.Background(), testIDDane)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	// Very flaky test?
	if bds != nil {
		require.True(t, len(bds) > 0)
	}
}

func TestGetAssetClassInfo(t *testing.T) {
	bds, err := GetAssetClassInfo(context.Background(), testAppTF2, []int{195151, 16891096})
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	require.NoError(t, err)
	require.NotNil(t, bds)
}

func TestGetGroupMembers(t *testing.T) {
	steamIds, err := GetGroupMembers(context.TODO(), 103582791429521412)
	if err != nil && errors.Is(err, ErrServiceUnavailable) {
		t.Skipf("Service not available currently")
		return
	}
	found := false
	for _, sid := range steamIds {
		if sid == 76561197985607672 {
			found = true
			break
		}
	}
	require.NoError(t, err)
	require.True(t, found)
}
