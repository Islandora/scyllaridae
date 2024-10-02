function decode_percent(s)
  return s:gsub('%%([0-9A-Fa-f][0-9A-Fa-f])', function(hex)
    return string.char(tonumber(hex, 16))
  end)
end

function Str(el)
  el.text = decode_percent(el.text)
  return el
end

function Code(el)
  el.text = decode_percent(el.text)
  return el
end
