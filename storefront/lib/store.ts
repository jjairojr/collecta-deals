export const STORE = {
  name: "Collecta TCG",
  legalName: "Collecta",
  street: "Rua 10, 163, Loja 03",
  building: "Galeria Badra",
  district: "Setor Oeste",
  city: "Goiânia",
  state: "GO",
  cep: "74120-020",
  country: "BR",
  phone: "+55 62 99339-9980",
  phoneDigits: "5562993399980",
  opens: "12:00",
  closes: "22:00",
  instagram: "https://www.instagram.com/tcgcollecta",
  instagramHandle: "@tcgcollecta",
} as const;

export const STORE_ADDRESS_LINE = `${STORE.street}, ${STORE.building}, ${STORE.district}, ${STORE.city} - ${STORE.state}, CEP ${STORE.cep}`;

const mapsQuery = encodeURIComponent(`Rua 10, 163, ${STORE.district}, ${STORE.city} - ${STORE.state}, ${STORE.cep}`);

export const STORE_MAPS_URL = `https://www.google.com/maps/search/?api=1&query=${mapsQuery}`;
export const STORE_MAPS_EMBED_URL = `https://www.google.com/maps?q=${mapsQuery}&output=embed`;
export const STORE_WHATSAPP_URL = `https://wa.me/${STORE.phoneDigits}`;

export const STORE_PHOTOS = [
  { src: "/loja/fachada.jpg", alt: "Fachada rosa da Collecta TCG na Galeria Badra, Setor Oeste, Goiânia", width: 957, height: 1200 },
  { src: "/loja/interior.jpg", alt: "Interior da Collecta TCG com vitrines de cartas, geladeira e mesas de jogo", width: 960, height: 1200 },
  { src: "/loja/balcao.jpg", alt: "Balcão da Collecta TCG com jogos de tabuleiro e área Play Game", width: 956, height: 1200 },
] as const;
