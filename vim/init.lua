vim.opt.compatible = false
vim.opt.number = true
vim.opt.laststatus = 2
vim.opt.statusline = "%f"
vim.opt.termguicolors = true

vim.cmd("syntax on")
vim.cmd("colorscheme default")

-- Necessary custom settings for some color schemes
vim.cmd("let g:solarized_termcolors=256")
vim.cmd("let g:zenbones_compat = 1")

vim.api.nvim_create_autocmd("BufReadPost", {
	pattern = "code_sample.vim",
	callback = function()
		require("extractor").extract(vim.env.COLOR_DATA_PATH)
	end,
})
