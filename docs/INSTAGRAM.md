# Instagram: a conta secundária (D40)

Reel público costuma baixar sem login. Quando o Instagram pede login (ele aperta de tempos em tempos), o
Acervo usa os **cookies de uma conta secundária**, guardados no servidor em
`/srv/data/acervo/instagram-cookies.txt`. Nunca use a conta principal: o Instagram pode bloquear uma
conta que baixa vídeos de forma automática.

## Pôr os cookies (ou renovar, quando vencerem)

O Acervo avisa na Referência: "O Instagram pediu login…" (faltam) ou "…recusou os cookies…" (venceram).

1. No Mac, abra o **Firefox** (ele fica só para a conta secundária; o Chrome continua com a principal).
2. Entre em <https://www.instagram.com> com a **conta secundária**. Se o Instagram pedir um código ou
   confirmação, faça. Abra um Reel qualquer para conferir que está logado.
3. **Feche o Firefox** (com ele aberto, o arquivo de cookies fica travado).
4. No Terminal, na pasta do Acervo:

   ```sh
   scripts/cookies-instagram.sh
   ```

   O script lê só os cookies do `instagram.com`, confere o login com um Reel e manda o arquivo ao
   servidor. Na primeira vez o macOS pode pedir permissão para o Terminal ler os dados do Firefox.
5. No Acervo, abra a Referência que deu erro e mande extrair de novo.

Outro navegador: `scripts/cookies-instagram.sh chrome:"Profile 1"` (o Chrome pede a senha do
Chaveiro) ou `scripts/cookies-instagram.sh safari`.

## Tirar os cookies

```sh
ssh lucas@192.168.1.20 rm /srv/data/acervo/instagram-cookies.txt
```

Sem o arquivo, o Acervo volta a baixar sem login.
