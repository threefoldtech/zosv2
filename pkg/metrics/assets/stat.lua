-- key id (A|D) value
-- 2 Keys and 2 ARGS

local key = KEYS[1]
local id = KEYS[2]
local type = ARGV[1]
local sample = tonumber(ARGV[2])

local statekey = string.format("metric:%s:%s", key, id)
local now = redis.call('TIME')[1]

-- local hsetkey = string.format("stats:%s", node)
local value = {}
local stat
local stored = redis.call('GET', statekey)

local m_timestamp = (math.floor(now / 300) * 300) + 300
local h_timestamp = (math.floor(now / 3600) * 3600) + 3600

local differential = type == "D"

if stored then
    value = cjson.decode(stored)
else
    value = {
        last_update = now,
        m_avg = 0,
        m_max = 0,
        m_total = 0,
        m_nr = 0,
        m_timestamp = m_timestamp,
        m_previous = {
            avg = 0,
            max = 0,
            timestamp = 0,
        },

        h_avg = 0,
        h_max = 0,
        h_total = 0,
        h_nr = 0,
        h_timestamp = h_timestamp,
        h_previous = {
            avg = 0,
            max = 0,
            timestamp = 0,
        },
    }
end

local input_sample = sample

if differential then
    local last = value.last
    if not stored then
        -- no previous value
        value.last = sample
        return
    end

    local diff = sample - last
    if now == value.last_update then
        -- what should we do here?
        return
    end
    input_sample = diff / (now - value.last_update)
    value.last = sample
end

value.last_update = now
-- the next 2 blocks pushes older value as previous for both
-- 5m and 1h samples

-- for 5m
if value.m_timestamp ~= m_timestamp then
    -- new value is in a new slot
    value.m_previous = {
        avg = value.m_avg,
        max = value.m_max,
        timestamp = value.m_timestamp,
    }

    value.m_timestamp = m_timestamp
    value.m_max = 0
    value.m_avg = 0
    value.m_total = 0
    value.m_nr = 0
end

-- for 1h
if value.h_timestamp ~= h_timestamp then
    -- new value is in a new slot
    value.h_previous = {
        avg = value.h_avg,
        max = value.h_max,
        timestamp = value.h_timestamp,
    }

    value.h_timestamp = h_timestamp
    value.h_max = 0
    value.h_avg = 0
    value.h_total = 0
    value.h_nr = 0
end

-- now update the new value
value.m_nr = value.m_nr + 1
value.m_total = value.m_total + input_sample
if input_sample > value.m_max then
    value.m_max = input_sample
end
value.m_avg = value.m_total / value.m_nr

value.h_nr = value.h_nr + 1
value.h_total = value.h_total + input_sample
if input_sample > value.h_max then
    value.h_max = input_sample
end
value.h_avg = value.h_total / value.h_nr

local data = cjson.encode(value)
redis.call('SET', statekey, data)
redis.call('EXPIRE', statekey, 24*60*60) -- expire in a day
