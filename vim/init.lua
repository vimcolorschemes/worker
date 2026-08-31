vim.opt.compatible = false
vim.opt.number = true
vim.opt.laststatus = 2
vim.opt.statusline = "%f"
vim.opt.termguicolors = true

vim.cmd("syntax on")
vim.cmd("colorscheme default")

-- Necessary custom settings for some colorschemes
vim.cmd("let g:solarized_termcolors=256")
