"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import Field from "@/components/checkout/Field";
import { useAccount } from "@/lib/account";
import { validEmail } from "@/lib/checkout";

export default function LoginForm({ next }: { next: string }) {
  const router = useRouter();
  const { refresh } = useAccount();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [notice, setNotice] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    const e: { email?: string; password?: string } = {};
    if (!validEmail(email)) {
      e.email = "E-mail inválido";
    }
    if (!password) {
      e.password = "Informe a senha";
    }
    setErrors(e);
    if (Object.keys(e).length > 0) {
      return;
    }
    setSubmitting(true);
    setNotice(null);
    try {
      const res = await fetch("/api/conta/entrar", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });
      const data = (await res.json()) as { error?: string };
      if (!res.ok) {
        setNotice(data.error ?? "Não foi possível entrar.");
        return;
      }
      await refresh();
      router.push(next);
    } catch {
      setNotice("Sem conexão. Tente de novo em instantes.");
    } finally {
      setSubmitting(false);
    }
  };

  const registerHref = next === "/conta" ? "/conta/criar" : `/conta/criar?next=${encodeURIComponent(next)}`;

  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <Field
        label="E-mail"
        name="email"
        value={email}
        onChange={(v) => {
          setEmail(v);
          setErrors((x) => ({ ...x, email: undefined }));
        }}
        error={errors.email}
        inputMode="email"
        autoComplete="email"
      />
      <PasswordField
        label="Senha"
        value={password}
        onChange={(v) => {
          setPassword(v);
          setErrors((x) => ({ ...x, password: undefined }));
        }}
        error={errors.password}
        autoComplete="current-password"
      />
      {notice && (
        <p role="alert" className="sticker rounded-[10px] bg-brand-soft px-3 py-2 text-xs font-semibold text-outline [--sh:3px]">
          {notice}
        </p>
      )}
      <button
        type="submit"
        disabled={submitting}
        className="arcade-press sticker mt-1 w-full rounded-[10px] bg-brand px-5 py-3.5 font-pixel text-[10px] text-white [--sh:5px] disabled:cursor-wait disabled:opacity-60"
      >
        {submitting ? "ENTRANDO..." : "ENTRAR"}
      </button>
      <p className="text-center text-xs text-muted">
        Ainda não tem conta?{" "}
        <Link href={registerHref} className="text-brand-soft underline-offset-4 hover:underline">
          Criar conta Collecta
        </Link>
      </p>
      <p className="text-center font-pixel text-[7px] leading-relaxed text-faint">
        MESMA CONTA DO APP COLLECTA
      </p>
    </form>
  );
}

export function PasswordField({
  label,
  value,
  onChange,
  error,
  autoComplete,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  error?: string;
  autoComplete: string;
}) {
  const [show, setShow] = useState(false);
  return (
    <label className="block min-w-0">
      <span className="mb-1.5 block font-pixel text-[8px] text-brand-soft">{label.toUpperCase()}</span>
      <span
        className={`sticker flex overflow-hidden rounded-[10px] border-[3px] bg-white [--sh:0px] ${error ? "border-brand" : ""}`}
      >
        <input
          type={show ? "text" : "password"}
          name="password"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          autoComplete={autoComplete}
          aria-label={label}
          aria-invalid={error ? true : undefined}
          className="min-w-0 flex-1 bg-transparent px-3 py-2.5 text-base text-outline outline-none placeholder:text-faint sm:text-sm"
        />
        <button
          type="button"
          onClick={() => setShow((s) => !s)}
          className="border-l-4 border-outline bg-brand-soft px-3 font-pixel text-[8px] text-outline"
          aria-label={show ? "Ocultar senha" : "Mostrar senha"}
        >
          {show ? "OCULTAR" : "VER"}
        </button>
      </span>
      {error && <span className="mt-1 block font-pixel text-[7px] text-brand">{error.toUpperCase()}</span>}
    </label>
  );
}
