package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// A capa da Referência na lista (D46). O vídeo do YouTube usa a miniatura do próprio YouTube (no
// frontend). O Reel guarda a capa do post (ou, sem ela, um quadro do vídeo) e o podcast, a capa do
// episódio, as duas reduzidas pelo ffmpeg para capa.jpg na pasta da Referência. Sem capa, a lista
// mostra o ícone do Instagram ou do Spotify. A capa é um enfeite: se falhar, a extração segue.

const larguraDaCapa = 320

var clienteDaCapa = &http.Client{Timeout: time.Minute}

func (a *app) arquivoDaCapa(id string) string {
	return filepath.Join(a.pastaReferencia(id), "capa.jpg")
}

func (a *app) temCapa(id string) bool {
	_, err := os.Stat(a.arquivoDaCapa(id))
	return err == nil
}

// gravarCapa reduz uma imagem, ou o quadro de 1 s de um vídeo, para a capa da Referência.
func (a *app) gravarCapa(ctx context.Context, id, origem string, video bool) error {
	destino := a.arquivoDaCapa(id)
	tmp := destino + ".tmp.jpg"
	args := []string{"-hide_banner", "-nostats", "-y"}
	if video {
		args = append(args, "-ss", "1")
	}
	args = append(args, "-i", origem, "-frames:v", "1", "-vf", fmt.Sprintf("scale='min(%d,iw)':-2", larguraDaCapa), "-q:v", "4", tmp)
	if out, err := exec.CommandContext(ctx, a.cfg.ffmpeg, args...).CombinedOutput(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("o ffmpeg não fez a capa: %v %s", err, cortar(string(out), 300, false))
	}
	return os.Rename(tmp, destino)
}

// capaDoReel: a miniatura que o yt-dlp trouxe com --write-thumbnail; sem ela, um quadro do vídeo.
func (a *app) capaDoReel(ctx context.Context, id, pasta string) error {
	for _, ext := range []string{"jpg", "jpeg", "webp", "png"} {
		if img := filepath.Join(pasta, "reel."+ext); existe(img) {
			if err := a.gravarCapa(ctx, id, img, false); err == nil {
				return nil
			}
		}
	}
	if video := filepath.Join(pasta, "reel.mp4"); existe(video) {
		return a.gravarCapa(ctx, id, video, true)
	}
	return errors.New("o Reel veio sem capa")
}

// capaDoPodcast baixa a capa do episódio (a do feed) e a reduz.
func (a *app) capaDoPodcast(ctx context.Context, id, endereco, pasta string) error {
	if endereco == "" {
		return errors.New("o episódio não tem capa no feed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Acervo-homelab")
	res, err := clienteDaCapa.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("a capa respondeu %d", res.StatusCode)
	}
	img := filepath.Join(pasta, "capa-original")
	f, err := os.Create(img)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, io.LimitReader(res.Body, 20<<20))
	f.Close()
	if err != nil {
		return err
	}
	return a.gravarCapa(ctx, id, img, false)
}

// handleCapa: GET /api/referencias/{id}/capa.
func (a *app) handleCapa(w http.ResponseWriter, r *http.Request) {
	ref, err := a.referencia(r.Context(), r.PathValue("id"))
	if err != nil || !a.temCapa(ref.ID) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, a.arquivoDaCapa(ref.ID))
}

// As Referências de antes da capa (D46): uma tarefa só, sem aviso, busca as que faltam. Quem não
// tem capa no Instagram ou no feed ganha capa.falta, para não tentar de novo a cada vez.
func (a *app) capasQueFaltam(ctx context.Context) []string {
	rows, err := a.db.QueryContext(ctx, `SELECT id FROM referencias WHERE tipo IN ('reel', 'podcast') ORDER BY criado_em DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		p := a.pastaReferencia(id)
		if existe(filepath.Join(p, "cru.md")) && !existe(filepath.Join(p, "capa.jpg")) && !existe(filepath.Join(p, "capa.falta")) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (a *app) pedirCapasQueFaltam(ctx context.Context) {
	if len(a.capasQueFaltam(ctx)) == 0 {
		return
	}
	var ja int
	a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tarefas WHERE tipo = 'buscar_capas' AND estado IN (?, ?)`, naFila, rodando).Scan(&ja)
	if ja == 0 {
		if _, err := a.fila.enfileirar(ctx, "", "buscar_capas", nil); err != nil {
			log.Printf("capas: %v", err)
		}
	}
}

func (a *app) buscarCapas(ctx context.Context, t *tarefa) (resultado, error) {
	for _, id := range a.capasQueFaltam(ctx) {
		if ctx.Err() != nil {
			return pronto, ctx.Err()
		}
		ref, err := a.referencia(ctx, id)
		if errors.Is(err, errNaoEncontrado) {
			continue
		} else if err != nil {
			return pronto, err
		}
		if err := a.buscarCapa(ctx, ref); err != nil {
			log.Printf("capa de %s: %v", id, err)
			os.WriteFile(filepath.Join(a.pastaReferencia(id), "capa.falta"), []byte(err.Error()+"\n"), 0o644)
		}
	}
	return pronto, nil
}

func (a *app) buscarCapa(ctx context.Context, ref *referencia) error {
	tmp, err := os.MkdirTemp("", "capa-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if ref.Tipo == "podcast" {
		var ep episodio
		b, err := os.ReadFile(filepath.Join(a.pastaReferencia(ref.ID), "info.json"))
		if err != nil || json.Unmarshal(b, &ep) != nil {
			return errors.New("o podcast não tem info.json")
		}
		return a.capaDoPodcast(ctx, ref.ID, ep.Capa, tmp)
	}
	if _, err := a.ytdlp(ctx, "-q", "--no-playlist", "--skip-download", "--write-thumbnail",
		"-o", filepath.Join(tmp, "reel.%(ext)s"), ref.URL); err != nil {
		return a.erroDoInstagram(err)
	}
	return a.capaDoReel(ctx, ref.ID, tmp)
}
