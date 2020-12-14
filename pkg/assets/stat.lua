--key,value,str(now),type,tags

local key = KEYS[1]
local value = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local type = ARGV[3]
local tags = ARGV[4]
local node = ARGV[5]

local statekey = string.format("stats:%s:%s", node, key)

-- local hsetkey = string.format("stats:%s", node)
local v = {}
local c = ""
local stat
local prev = redis.call('GET', statekey)

local now_short_m = (math.floor(now / 300) * 300) + 300
local now_short_h = (math.floor(now / 3600) * 3600) + 3600

local differential = type == "D"

if prev then
    -- get previous value, it exists in a hkey
    v = cjson.decode(prev)
    local diff
    local difftime

    -- calculate the dif when absolute nrs are logged e.g. byte counter for network
    if differential then
        -- diff
        diff = value - v.val
        difftime = now - v.epoch
        -- calculate diff per second.
        stat = diff / difftime
    else
        stat = value
    end

    -- Check if we can flush the previous aggregated values.

    if v.m_epoch < now_short_m then
        -- 1 min aggregation
        local row = string.format("%s|%s|%u|%f|%f|%f|%f",
            node, key, v.m_epoch, stat, v.m_avg, v.m_max, v.m_total)

        redis.call("RPUSH", "queues:stats:min", row)

        v.m_total = 0
        v.m_nr = 0
        v.m_max = value
        v.m_epoch = now_short_m
    end
    if v.h_epoch < now_short_h then
        -- 1 hour aggregation
        local row = string.format("%s|%s|%u|%f|%f|%f|%f",
            node, key, v.h_epoch, stat, v.h_avg, v.h_max, v.h_total)

        redis.call("RPUSH", "queues:stats:hour", row)

        v.h_total = 0
        v.h_nr = 0
        v.h_max = value
        v.h_epoch = now_short_h
    end

    -- remember the current value
    v.val = value
    v.epoch = now

    --remember current measurement, and calculate the avg/max for minute value
    v.m_last = stat
    v.m_total = v.m_total + stat
    v.m_nr = v.m_nr + 1
    v.m_avg = v.m_total / v.m_nr
    if stat > v.m_max then
        v.m_max = stat
    end

    -- work for the hour period
    --h_last is not required would not provide additional info
    v.h_total = v.h_total + stat
    v.h_nr = v.h_nr + 1
    v.h_avg = v.h_total / v.h_nr
    if stat > v.h_max then
        v.h_max = stat
    end

    -- always reset tags in case of change.
    v.tags = tags
    -- remember in redis
    local data = cjson.encode(v)
    redis.call('SET', statekey, data)
    redis.call('EXPIRE', statekey, 24*60*60) -- expire in a day

    -- don't grow over 200,000 records
    redis.call("LTRIM", "queues:stats:min", -200000, -1)
    redis.call("LTRIM", "queues:stats:hour", -200000, -1)

    return data
else
    if differential then
        --differential stats
        v.m_avg = 0
        v.m_total = 0
        v.m_max = 0
        v.m_nr = 0

        v.h_avg = 0
        v.h_total = 0
        v.h_max = 0
        v.h_nr = 0
    else
        --gauages stats
        v.m_avg = value
        v.m_total = value
        v.m_max = value
        v.m_nr = 1

        v.h_avg = value
        v.h_total = value
        v.h_max = value
        v.h_nr = 1
    end

    v.m_last = 0
    v.m_epoch = now_short_m
    v.h_epoch = now_short_h

    v.epoch = now
    v.val = value
    v.key = key
    v.tags = tags

    local data = cjson.encode(v)
    redis.call('SET', statekey, data)
    redis.call('EXPIRE', statekey, 24*60*60) -- expire in a day
    return data
end
