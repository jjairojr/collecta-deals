import type { ReactNode } from "react";
import { AlertTriangle, ExternalLink, Info, Lightbulb } from "lucide-react";

const LOJA = "866280";
const ADMIN = "https://www.ligamagic.com.br/?view=ecom/admin";
const LOJA_URL = `https://www.ligamagic.com.br/?view=ecom/home&id=${LOJA}`;

function Section({ n, title, children }: { n: string; title: string; children: ReactNode }) {
  return (
    <section className="mt-10">
      <div className="flex items-center gap-3">
        <span className="sticker sticker-sm font-pixel flex h-8 w-8 shrink-0 items-center justify-center rounded-[8px] bg-brand text-[10px] text-white">
          {n}
        </span>
        <h2 className="font-display text-xl font-extrabold text-fg">{title}</h2>
      </div>
      <div className="mt-4 space-y-3 text-sm leading-relaxed text-slate-300">{children}</div>
    </section>
  );
}

function Shot({ src, alt, caption }: { src: string; alt: string; caption: string }) {
  return (
    <figure className="mt-4">
      <a href={src} target="_blank" rel="noreferrer" className="block">
        <img
          src={src}
          alt={alt}
          className="sticker w-full rounded-[10px] border-outline transition-opacity hover:opacity-90"
        />
      </a>
      <figcaption className="font-pixel mt-2 text-[9px] leading-relaxed text-slate-500">
        {caption} — clique para abrir em tamanho real
      </figcaption>
    </figure>
  );
}

function Callout({ kind, children }: { kind: "aviso" | "dica" | "nota"; children: ReactNode }) {
  const style =
    kind === "aviso"
      ? { border: "border-loss", bg: "bg-loss/10", Icon: AlertTriangle, tone: "text-loss" }
      : kind === "dica"
        ? { border: "border-gain", bg: "bg-gain/10", Icon: Lightbulb, tone: "text-gain" }
        : { border: "border-royal-light", bg: "bg-royal/15", Icon: Info, tone: "text-royal-light" };
  const { Icon } = style;
  return (
    <div className={`flex gap-3 rounded-[10px] border-[3px] ${style.border} ${style.bg} p-3`}>
      <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${style.tone}`} />
      <div className="text-sm leading-relaxed text-slate-200">{children}</div>
    </div>
  );
}

function Steps({ items }: { items: ReactNode[] }) {
  return (
    <ol className="space-y-2">
      {items.map((item, i) => (
        <li key={i} className="flex gap-3">
          <span className="font-pixel mt-0.5 w-5 shrink-0 text-[10px] text-brand-label">{i + 1}.</span>
          <span>{item}</span>
        </li>
      ))}
    </ol>
  );
}

function Key({ children }: { children: ReactNode }) {
  return (
    <span className="rounded-[5px] border-[2px] border-outline bg-raised px-1.5 py-0.5 font-mono text-[12px] text-fg">
      {children}
    </span>
  );
}

function Link({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="inline-flex items-center gap-1 font-bold text-brand-label underline decoration-brand/50 underline-offset-2 hover:text-brand"
    >
      {children}
      <ExternalLink className="h-3 w-3" />
    </a>
  );
}

function Table({ head, rows }: { head: string[]; rows: string[][] }) {
  return (
    <div className="sticker overflow-x-auto rounded-[10px] bg-panel">
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b-[3px] border-outline bg-raised">
            {head.map((h) => (
              <th key={h} className="font-pixel px-3 py-2 text-[9px] whitespace-nowrap text-brand-label">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i} className="border-b border-slate-800 last:border-0">
              {r.map((c, j) => (
                <td key={j} className={`px-3 py-2 ${j === 0 ? "font-bold text-fg" : "text-slate-300"}`}>
                  {c}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function GuiaLigaPage() {
  return (
    <div className="max-w-4xl">
      <div className="sticker rounded-[12px] bg-panel p-5">
        <p className="text-sm leading-relaxed text-slate-200">
          Tudo que a loja da Liga precisa no dia a dia, sem depender de ninguém. Todos os prints são da nossa
          loja de verdade (id <Key>{LOJA}</Key>). Entre em{" "}
          <Link href={`${ADMIN}/home`}>Dashboard Administrativo</Link> e a loja pública fica em{" "}
          <Link href={LOJA_URL}>ligamagic.com.br</Link>.
        </p>
      </div>

      <Section n="01" title="Como a nossa loja está organizada">
        <p>
          Três controles diferentes decidem se um jogo ou produto aparece. Eles são{" "}
          <strong className="text-fg">independentes</strong> — é aqui que mora quase todo problema:
        </p>
        <Steps
          items={[
            <>
              <strong className="text-fg">Exibição de Cartas</strong> (Layout → Preferências da Loja Virtual):
              se estiver em <em>Nenhuma</em>, o jogo fica privado e nenhuma carta aparece.
            </>,
            <>
              <strong className="text-fg">Categoria</strong> (Cards e Produtos → Categorias): é isso que{" "}
              <em>cria o item de menu</em>. Jogo público sem categoria não é alcançável pelo menu.
            </>,
            <>
              <strong className="text-fg">Plugin de Marketplace</strong> (Minha Conta → Assinatura): é o que
              indexa nosso estoque no comparador da Liga. Não tem efeito na loja em si.
            </>,
          ]}
        />
        <Callout kind="nota">
          O menu <strong>Produtos Selados</strong> é organizado <strong>por jogo</strong> (One Piece, Pokémon,
          Riftbound), não por tipo de produto. As ~39 subcategorias de tipo (Caixas de Boosters, Blisters,
          Secret Lair…) estão <strong>ocultas</strong> de propósito.
        </Callout>
        <Shot
          src="/guia-liga/06-menu-loja.jpg"
          alt="Menu da loja com o submenu de Produtos Selados aberto"
          caption="O menu como o cliente vê"
        />
      </Section>

      <Section n="02" title="Cadastrar um produto selado">
        <p>
          A Liga <strong className="text-fg">já tem o catálogo pronto</strong>, com nome, foto e ficha. Nunca
          use “Novo Produto” para algo que já existe: a gente só encontra o produto e informa quanto tem e por
          quanto vende.
        </p>
        <p>
          Vá em <Link href={`${ADMIN}/prod/all`}>Cards e Produtos → Cadastrar Produtos</Link> e:
        </p>
        <Steps
          items={[
            <>
              Em <strong className="text-fg">Tipo de Cadastro</strong> marque{" "}
              <Key>Base interna do Sistema</Key> (não “Meu Cadastro”, que só mostra o que já é nosso).
            </>,
            <>
              <strong className="text-fg">Tipo de Produto</strong>: <Key>Produto Selado (Card Game)</Key>.
            </>,
            <>
              <strong className="text-fg">Card Game</strong>: o jogo. E em <strong className="text-fg">Produto</strong>,
              o miolo do nome — <Key>Vendetta</Key>, não “Caixa de Booster - Vendetta”.
            </>,
            <>Clique em Buscar. Na linha certa, preencha Estoque, Preço, Idioma e Condição.</>,
            <>
              Clique em <strong className="text-fg">Salvar</strong>. Dá para preencher várias linhas e salvar
              todas de uma vez.
            </>,
          ]}
        />
        <Shot
          src="/guia-liga/01-cadastrar.jpg"
          alt="Tela de cadastro de produtos com a busca na base do sistema"
          caption="Busca na base do sistema e as colunas de preenchimento"
        />
        <Callout kind="aviso">
          A Liga traduz os nomes. Procure <strong>Deck Inicial</strong> (não “Starter Deck”),{" "}
          <strong>Coleção Treinador Avançado</strong> (não “Elite Trainer Box”),{" "}
          <strong>Combo de Pacotes</strong> (não “Booster Bundle”) e <strong>Caixa de Booster</strong> (não
          “Booster Box”). Buscar em inglês não acha nada.
        </Callout>
        <Callout kind="aviso">
          Confira o <strong>idioma no nome</strong>. A busca devolve <Key>(ING)</Key> e <Key>(PT-BR)</Key> lado
          a lado com nome quase igual. Escolher o errado cadastra o produto inglês com estoque nacional. E leia
          o nome até o fim: às vezes existe uma variante <em>“- Pokémon Center”</em>.
        </Callout>
      </Section>

      <Section n="03" title="Depois de cadastrar, mover para a categoria do jogo">
        <Callout kind="aviso">
          <strong>Esse passo é obrigatório.</strong> A Liga joga todo produto novo numa categoria de{" "}
          <em>tipo</em> (Caixas de Boosters, Blisters…), e essas categorias estão ocultas. Se você não mover, o
          produto <strong>não aparece na loja</strong> e ninguém percebe.
        </Callout>
        <Steps
          items={[
            <>
              Na linha do produto (em <Key>Meu Cadastro</Key>), clique no ícone de{" "}
              <strong className="text-fg">lápis</strong> (Editar produto).
            </>,
            <>
              No topo, troque o campo <strong className="text-fg">Categoria</strong> para o jogo:{" "}
              <Key>.. One Piece</Key>, <Key>.. Pokémon</Key> ou <Key>.. Riftbound</Key>.
            </>,
            <>
              Clique em <strong className="text-fg">Salvar Produto</strong>. Não mexa em mais nada.
            </>,
          ]}
        />
        <Shot
          src="/guia-liga/04-mover-categoria.jpg"
          alt="Tela Editar Produto com o campo Categoria"
          caption="Editar Produto — só o campo Categoria precisa mudar"
        />
        <Callout kind="nota">
          Essa tela é a <strong>nossa listagem</strong>, não o catálogo compartilhado da Liga (ela tem nosso
          preço, idioma, SKU e dados fiscais). Mudar a categoria não afeta outras lojas.
        </Callout>
      </Section>

      <Section n="04" title="Mudar preço ou quantidade">
        <p>
          Em <Link href={`${ADMIN}/prod/all`}>Cadastrar Produtos</Link>, marque{" "}
          <Key>Meu Cadastro</Key>, busque o produto e edite Estoque ou Preço direto na linha. Depois{" "}
          <strong className="text-fg">Salvar</strong>.
        </p>
        <Shot
          src="/guia-liga/02-meu-cadastro.jpg"
          alt="Listagem Meu Cadastro com estoque e preço editáveis"
          caption="Meu Cadastro — estoque e preço editáveis na própria linha"
        />
        <Callout kind="aviso">
          A linha cinza do topo aplica valores em <strong>todas</strong> as linhas de uma vez. Deixe o{" "}
          <Key>Somente campos sem preenchimento</Key> sempre <strong>marcado</strong>. Desmarcado, com os
          campos do topo vazios, um Salvar <strong>zera o estoque de tudo</strong>.
        </Callout>
        <Callout kind="dica">
          Preço usa vírgula: <Key>39,90</Key>. Digitando <Key>130</Key> ele completa <Key>130,00</Key> sozinho.
        </Callout>
      </Section>

      <Section n="05" title="Tirar um produto da loja">
        <p>Duas formas, para situações diferentes:</p>
        <Steps
          items={[
            <>
              <strong className="text-fg">Acabou o estoque (vai voltar):</strong> coloque o estoque em{" "}
              <Key>0</Key> e salve. Como os três jogos estão em “Somente com Estoque”, ele sai da loja
              sozinho e o cadastro fica pronto para quando chegar mais.
            </>,
            <>
              <strong className="text-fg">Não vendemos mais:</strong> no menu <Key>⋮</Key> da linha, use{" "}
              <strong className="text-fg">Remover Produto</strong>.
            </>,
          ]}
        />
        <Shot
          src="/guia-liga/03-remover-produto.jpg"
          alt="Menu de três pontos aberto com a opção Remover Produto"
          caption="O menu ⋮ da linha — Remover Produto é o último item"
        />
        <Callout kind="dica">
          Na dúvida, prefira zerar o estoque. É reversível e preserva o preço, o idioma e a categoria que a
          gente já ajustou.
        </Callout>
      </Section>

      <Section n="06" title="Categorias: editar, ocultar e apagar">
        <p>
          Em <Link href={`${ADMIN}/categorias`}>Cards e Produtos → Categorias</Link>, cada linha tem até três
          ícones:
        </p>
        <Table
          head={["Ícone", "O que faz", "Reversível?"]}
          rows={[
            ["✎ lápis", "Editar nome, tipo, regra e posição", "sim"],
            ["✕ vermelho", "Apagar a categoria de vez", "NÃO"],
            ["👁 olho verde", "Ocultar / exibir no menu da loja", "sim"],
          ]}
        />
        <Shot
          src="/guia-liga/05-categorias.jpg"
          alt="Lista de categorias com os ícones de ação"
          caption="Repare: as categorias de carta têm só 2 ícones; as outras têm o olho"
        />
        <Callout kind="aviso">
          O <strong>✕ apaga na hora, sem pedir confirmação</strong>. Se quer só tirar do menu, use o olho.
        </Callout>
        <Callout kind="nota">
          Categorias de card game (Cartas de One Piece, Cartas de Pokémon…) <strong>não têm o olho</strong> —
          não existe como ocultá-las, só apagar. É por isso que Magic e Yugioh continuam no menu.
        </Callout>
        <Callout kind="nota">
          Não dá para apagar uma categoria que tem subcategorias ativas. Apague as filhas primeiro.
        </Callout>
        <p className="pt-1">
          Ao criar uma categoria, o campo <strong className="text-fg">Regra Específica</strong> decide o que ela
          lista: <Key>Card Game</Key> lista <strong>cartas</strong>, <Key>Sem Regra</Key> lista{" "}
          <strong>produtos</strong>. Nossas três subs de selado são <em>Sem Regra</em> — com Card Game elas
          apareceriam vazias.
        </p>
      </Section>

      <Section n="07" title="Conferir se ficou certo">
        <p>
          Abra a categoria do jogo na loja pública. Ela mostra foto, bandeira do idioma, quantidade e preço:
        </p>
        <Table
          head={["Categoria na loja", "Link"]}
          rows={[
            ["Produtos Selados → One Piece", "?view=ecom/itens&id=866280&cat=256333"],
            ["Produtos Selados → Pokémon", "?view=ecom/itens&id=866280&cat=256334"],
            ["Produtos Selados → Riftbound", "?view=ecom/itens&id=866280&cat=256335"],
          ]}
        />
        <Shot
          src="/guia-liga/07-categoria-publica.jpg"
          alt="Categoria de selados de Pokémon na loja pública"
          caption="Selados de Pokémon na loja — 5 produtos com foto, idioma, quantidade e preço"
        />
        <Callout kind="dica">
          Página vazia quase sempre significa produto sem estoque ou ainda na categoria de tipo (oculta) — não
          é layout quebrado.
        </Callout>
        <Callout kind="nota">
          Logado como dono você vê categorias que o cliente não vê. Para checar a visão real do cliente, abra
          a loja numa <strong>janela privada</strong>.
        </Callout>
      </Section>

      <Section n="08" title="Colar na memória">
        <p className="font-bold text-fg">Idioma</p>
        <Table
          head={["Código", "Idioma"]}
          rows={[
            ["2", "Inglês — é o dos produtos marcados (ING)"],
            ["8", "Português — os (PT-BR)"],
            ["11", "Português / Inglês"],
            ["6", "Japonês"],
          ]}
        />
        <p className="pt-2 font-bold text-fg">Condição</p>
        <Table
          head={["Código", "Condição"]}
          rows={[
            ["2", "Lacrado — use este em todo produto selado"],
            ["3", "Novo"],
            ["4", "Novo com embalagem aberta"],
            ["6", "Usado"],
          ]}
        />
        <p className="pt-2 font-bold text-fg">Telas que mais usamos</p>
        <Table
          head={["Para quê", "Onde"]}
          rows={[
            ["Cadastrar / editar produto", "Cards e Produtos → Cadastrar Produtos"],
            ["Categorias e menu", "Cards e Produtos → Categorias"],
            ["Jogo público ou privado", "Layout → Preferências da Loja Virtual"],
            ["Banners e cores", "Layout → Configurar Layout Responsivo"],
            ["Blocos e notícias da home", "Layout → Página Inicial"],
            ["Pagamento e envio", "Configurações → Formas de Pagamento / Envio"],
            ["Plano e plugins", "Minha Conta → Assinatura"],
          ]}
        />
      </Section>

      <Section n="09" title="Onde é fácil errar">
        <Steps
          items={[
            <>
              Cadastrou e <strong className="text-fg">não moveu para a categoria do jogo</strong> → produto
              invisível na loja.
            </>,
            <>
              Escolheu o produto <Key>(ING)</Key> tendo estoque <Key>(PT-BR)</Key> → cliente recebe a versão
              errada.
            </>,
            <>
              Desmarcou o <Key>Somente campos sem preenchimento</Key> → risco de zerar o estoque de todas as
              linhas ao salvar.
            </>,
            <>
              Clicou no <Key>✕</Key> da categoria querendo só esconder → apagou, sem confirmação e sem volta.
            </>,
            <>
              Criou subcategoria de selado com regra <Key>Card Game</Key> → ela aparece vazia, porque Card Game
              lista cartas e não produtos.
            </>,
          ]}
        />
      </Section>

      <div className="sticker mt-10 rounded-[12px] bg-panel p-5">
        <p className="font-pixel text-[9px] leading-relaxed text-slate-500">
          A parte de layout e identidade visual da loja (logo, cores, banners, favicons) fica em
          storefront/design/liga no repositório, com o gerador dos arquivos nas medidas exatas que a Liga
          exige.
        </p>
      </div>
    </div>
  );
}
