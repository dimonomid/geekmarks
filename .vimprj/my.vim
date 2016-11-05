
" path to the .vimprj folder
let s:sVimprjPath  = expand('<sfile>:p:h')
let s:sPath  = simplify(s:sVimprjPath.'/..')

set tabstop=2
set shiftwidth=2

set colorcolumn=81

let g:indexer_indexerListFilename = s:sVimprjPath.'/.indexer_files'
let g:indexer_disableCtagsWarning = 1

" simplify tags somewhat (not so much)
let g:indexer_ctagsCommandLineOptions = '--c++-kinds=+p+l'
