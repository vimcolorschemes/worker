" Get the color group name of the syn ID
function! GetColorGroupName(synID) abort
  let l:name = synIDattr(a:synID, 'name')
  if l:name == ''
    let l:name = 'NormalFg'
  endif
  return l:name
endfunction

" Get the color group value of the syn ID
function! GetColorValue(synID) abort
  let l:color = synIDattr(synIDtrans(a:synID), 'fg#')
  if l:color == ''
    let l:color = synIDattr(hlID('Normal'), 'fg#')
  endif
  return l:color
endfunction

" Get some color values that are not picked up by GetColorValues
function! GetExtraColorValues() abort
  return {
        \ 'NormalFg':        synIDattr(hlID('Normal'),       'fg#'),
        \ 'NormalBg':        synIDattr(hlID('Normal'),       'bg#'),
        \ 'StatusLineFg':    synIDattr(hlID('StatusLine'),   'fg#'),
        \ 'StatusLineBg':    synIDattr(hlID('StatusLine'),   'bg#'),
        \ 'CursorFg':        synIDattr(hlID('Cursor'),       'fg#'),
        \ 'CursorBg':        synIDattr(hlID('Cursor'),       'bg#'),
        \ 'LineNrFg':        synIDattr(hlID('LineNr'),       'fg#'),
        \ 'LineNrBg':        synIDattr(hlID('LineNr'),       'bg#'),
        \ 'CursorLineFg':    synIDattr(hlID('CursorLine'),   'fg#'),
        \ 'CursorLineBg':    synIDattr(hlID('CursorLine'),   'bg#'),
        \ 'CursorLineNrFg':  synIDattr(hlID('CursorLineNr'), 'fg#'),
        \ 'CursorLineNrBg':  synIDattr(hlID('CursorLineNr'), 'bg#'),
        \  }
endfunction

" Get the last line # of the entire file
function! GetLastLine() abort
  return line('$')
endfunction

" Get the last column # of the given line
function! GetLastCol(line) abort
  call cursor(a:line, 1)
  return col('$')
endfunction

" Get color values of all words in the file + some more
function! GetColorValues() abort
  let l:lastline = GetLastLine()

  let l:currentline = 1

  let l:values = {}
  while l:currentline <= l:lastline
    let l:lastcol = GetLastCol(l:currentline)
    let l:currentcol = 1
    while l:currentcol <= l:lastcol
      call cursor(l:currentline, l:currentcol)

      let l:synID = synID(line('.'), col('.'), 1)
      let l:values[GetColorGroupName(l:synID)] = GetColorValue(l:synID)

      let l:currentcol += 1
    endwhile
    let l:currentline += 1
  endwhile

  call extend(l:values, GetExtraColorValues())

  return l:values
endfunction

" Returns true if the color hex value is considered to be light
function! IsHexColorLight(color) abort
  let l:rawColor = trim(a:color, '#')

  let l:red = str2nr(substitute(l:rawColor, '\(.\{2\}\).\{4\}', '\1', 'g'), 16)
  let l:green = str2nr(substitute(l:rawColor, '.\{2\}\(.\{2\}\).\{2\}', '\1', 'g'), 16)
  let l:blue = str2nr(substitute(l:rawColor, '.\{4\}\(.\{2\}\)', '\1', 'g'), 16)

  let l:brightness = ((l:red * 299) + (l:green * 587) + (l:blue * 114)) / 1000

  if l:brightness > 155
    return 1
  else
    return 0
  endif
endfunction

" Gets all color values of the current file and stores them in a file as JSON
function WriteColorValues(filename, background) abort
  let l:colorscheme = trim(execute('colorscheme'))
  if l:colorscheme != 'default'
    try
      let l:foreground = synIDattr(hlID('Normal'), 'fg#')

      let l:iscolorschemedark = IsHexColorLight(l:foreground)

      let l:data = {}
      if !l:iscolorschemedark && a:background == 'light' || l:iscolorschemedark && a:background == 'dark'
        let l:data = GetColorValues()
      endif

      let l:encoded_data = json_encode(l:data)
      call writefile([l:encoded_data], a:filename)
    catch
      echo 'error'
    endtry
  endif
endfunction
