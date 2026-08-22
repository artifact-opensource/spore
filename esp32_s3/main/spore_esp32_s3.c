#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#if __has_include(<lua.h>)
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#define SPORE_HAS_LUA 1
#else
#define SPORE_HAS_LUA 0
#endif

#if __has_include("esp_http_server.h")
#include "esp_http_server.h"
#define SPORE_HAS_HTTPD 1
#else
#define SPORE_HAS_HTTPD 0
#endif

#if __has_include("esp_now.h")
#include "esp_now.h"
#define SPORE_HAS_ESPNOW 1
#else
#define SPORE_HAS_ESPNOW 0
#endif

#define SPORE_RGB_GPIO 48

typedef struct {
    uint8_t r;
    uint8_t g;
    uint8_t b;
    char pattern[16];
} spore_rgb_t;

static const char *default_profile = "spore-default";

static spore_rgb_t fallback_rgb(const char *activity) {
    spore_rgb_t rgb = {0, 255, 136, "solid"};
    if (strcmp(activity, "thinking") == 0) {
        rgb = (spore_rgb_t){0, 212, 255, "pulse"};
    } else if (strcmp(activity, "tool") == 0) {
        rgb = (spore_rgb_t){255, 176, 0, "chase"};
    } else if (strcmp(activity, "error") == 0) {
        rgb = (spore_rgb_t){255, 51, 85, "blink"};
    } else if (strcmp(activity, "flash") == 0) {
        rgb = (spore_rgb_t){255, 68, 255, "pulse"};
    }
    return rgb;
}

static bool load_rgb_from_lua(const char *lua_path, const char *profile, const char *activity, spore_rgb_t *out) {
#if SPORE_HAS_LUA
    lua_State *L = luaL_newstate();
    if (!L) return false;
    luaL_openlibs(L);
    if (luaL_dofile(L, lua_path) != LUA_OK) {
        lua_close(L);
        return false;
    }
    lua_getglobal(L, "profiles");
    if (!lua_istable(L, -1)) {
        lua_close(L);
        return false;
    }
    lua_getfield(L, -1, profile);
    if (!lua_istable(L, -1)) {
        lua_close(L);
        return false;
    }
    lua_getfield(L, -1, activity);
    if (!lua_istable(L, -1)) {
        lua_close(L);
        return false;
    }
    lua_getfield(L, -1, "r"); out->r = (uint8_t)lua_tointeger(L, -1); lua_pop(L, 1);
    lua_getfield(L, -1, "g"); out->g = (uint8_t)lua_tointeger(L, -1); lua_pop(L, 1);
    lua_getfield(L, -1, "b"); out->b = (uint8_t)lua_tointeger(L, -1); lua_pop(L, 1);
    lua_getfield(L, -1, "pattern");
    snprintf(out->pattern, sizeof(out->pattern), "%s", lua_tostring(L, -1));
    lua_pop(L, 1);
    lua_close(L);
    return true;
#else
    (void)lua_path; (void)profile; (void)activity; (void)out;
    return false;
#endif
}

static spore_rgb_t resolve_rgb(const char *lua_path, const char *profile, const char *activity) {
    spore_rgb_t rgb = fallback_rgb(activity);
    if (lua_path && profile) {
        load_rgb_from_lua(lua_path, profile, activity, &rgb);
    }
    return rgb;
}

static void emit_status_snapshot(const char *profile, const char *activity, const spore_rgb_t *rgb) {
    printf("{\"profile\":\"%s\",\"activity\":\"%s\",\"gpio\":%d,\"rgb\":[%u,%u,%u],\"pattern\":\"%s\",\"espnow\":%s,\"httpd\":%s}\n",
           profile,
           activity,
           SPORE_RGB_GPIO,
           rgb->r,
           rgb->g,
           rgb->b,
           rgb->pattern,
           SPORE_HAS_ESPNOW ? "true" : "false",
           SPORE_HAS_HTTPD ? "true" : "false");
}

#if SPORE_HAS_HTTPD
static esp_err_t status_handler(httpd_req_t *req) {
    char body[96];
    snprintf(body, sizeof(body), "{\"service\":\"spore-command-centre\",\"gpio_rgb\":%d,\"espnow\":true}", SPORE_RGB_GPIO);
    httpd_resp_set_type(req, "application/json");
    return httpd_resp_sendstr(req, body);
}

static void start_http_server(void) {
    httpd_config_t config = HTTPD_DEFAULT_CONFIG();
    httpd_handle_t server = NULL;
    if (httpd_start(&server, &config) == ESP_OK) {
        httpd_uri_t uri = {
            .uri = "/api/status",
            .method = HTTP_GET,
            .handler = status_handler,
            .user_ctx = NULL,
        };
        httpd_register_uri_handler(server, &uri);
    }
}
#else
static void start_http_server(void) {
    printf("http server scaffold active\n");
}
#endif

static void init_espnow(void) {
#if SPORE_HAS_ESPNOW
    esp_now_init();
#else
    printf("espnow scaffold active\n");
#endif
}

#ifndef SPORE_HOST_SIM
void app_main(void) {
    init_espnow();
    start_http_server();
    spore_rgb_t rgb = resolve_rgb("/spiffs/rgb_profiles.lua", default_profile, "idle");
    emit_status_snapshot(default_profile, "idle", &rgb);
}
#else
int main(int argc, char **argv) {
    const char *lua_path = argc > 1 ? argv[1] : "main/rgb_profiles.lua";
    const char *activities[] = {"idle", "thinking", "tool", "error", "flash"};
    init_espnow();
    start_http_server();
    for (size_t i = 0; i < sizeof(activities) / sizeof(activities[0]); ++i) {
        spore_rgb_t rgb = resolve_rgb(lua_path, default_profile, activities[i]);
        emit_status_snapshot(default_profile, activities[i], &rgb);
    }
    return 0;
}
#endif
