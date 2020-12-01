function! s:markExpressionLimits()
    let l:notOnExpression = "ERROR: Cursor not on a gcd expression"

    if (search(s:EXP_PREFIX_PATTERN, "sbc", line(".")) == 0)
        throw l:notOnExpression
    endif

    if (search(s:EXP_PATTERN, "se", line(".")) == 0)
        throw l:notOnExpression
    endif
endfunction
