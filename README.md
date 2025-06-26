# Terminal Editor (Go)

Um simples editor de texto interativo para terminal, escrito em Go, com suporte a edição, salvamento, desfazer/refazer, e busca de texto.

> Desenvolvido com a biblioteca [tcell](https://github.com/gdamore/tcell) para manipulação de terminal.

# Funcionalidades

- Edição de texto no terminal
- Salvamento de arquivos (`Ctrl+S`)
- Sair do editor (`Ctrl+Q`)
- Undo/Redo (`Ctrl+Z` / `Ctrl+Y`)
- Busca interativa (`Ctrl+F`)
- Próximo resultado da busca (`Ctrl+N`)
- Multiplas linhas, novo parágrafo (`Enter`)
- Histórico de alterações persistente em memória
