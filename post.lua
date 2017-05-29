counter = 1
tags = 10000

request = function()
	wrk.method = "POST"
	vals = table.concat({tostring(math.random()), tostring(math.random()), tostring(math.random()), tostring(math.random()), tostring(math.random())}, ",")
	vals = vals .. "," .. table.concat({tostring(math.random()), tostring(math.random()), tostring(math.random()), tostring(math.random()), tostring(math.random())}, ",")
	time = tostring(os.time()) .. string.format("%u", 10^8 + counter)
	tag = "t" .. math.random(tags)

 	wrk.body   = string.format('{"time":%s,"tag":"%s","values":[%s]}', time, tag, vals)
	wrk.headers["Content-Type"] = "application/json"

	counter = counter + 1

	return wrk.format()
end
