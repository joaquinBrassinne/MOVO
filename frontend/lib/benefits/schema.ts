// Validación del formulario de beneficio con las mismas reglas que el dominio del backend
// (SDD §5.2). Es una ayuda para el usuario: la API vuelve a validar y manda.

import { z } from "zod";

import { parsePrice } from "../business/price.ts";

import {
  BENEFIT_TYPE,
  MAX_CONDITIONS_LENGTH,
  MAX_PERCENT,
  MAX_TITLE_LENGTH,
  type BenefitInput,
  type BenefitType,
} from "./types.ts";

// Textos planos: sin una salida de MOVO. Mismas expresiones que `domain/text.go`.
const LINK =
  /(?:https?:|ftp:|\/\/|www\.|\b[\p{L}\d-]+\.(?:com|net|org|info|biz|edu|gov|gob|io|ly|me|co|app|ar|us|es|cl|uy|br|xyz|site|online|shop|store|link)\b)/iu;
// Ocho dígitos o más, con un separador entre cada uno: el teléfono local más corto.
// Un precio como "$10.000" tiene cinco y pasa.
const PHONE = /\d(?:[\s().-]?\d){7,}/;

const MESSAGES = {
  percent: "Ingresá un porcentaje entero de 1 a 100",
  amount: "Ingresá un monto en pesos mayor que cero",
  titleRequired: "Escribí el título del beneficio",
  titleLong: `El título puede tener hasta ${MAX_TITLE_LENGTH} caracteres`,
  conditionsLong: `Las condiciones pueden tener hasta ${MAX_CONDITIONS_LENGTH} caracteres`,
  link: "No incluyas links",
  phone: "No incluyas teléfonos",
  control: "Usá texto simple, sin saltos de línea",
} as const;

/** Un texto plano ya recortado: sin links, sin teléfonos y sin caracteres de control. */
function plain(max: number, tooLong: string) {
  return z
    .string()
    .trim()
    .max(max, { error: tooLong })
    .refine((text) => !/\p{Cc}/u.test(text), { error: MESSAGES.control })
    .refine((text) => !LINK.test(text), { error: MESSAGES.link })
    .refine((text) => !PHONE.test(text), { error: MESSAGES.phone });
}

const conditions = plain(MAX_CONDITIONS_LENGTH, MESSAGES.conditionsLong);

const percentSchema = z.object({
  type: z.literal(BENEFIT_TYPE.PERCENT),
  percent: z
    .number({ error: MESSAGES.percent })
    .int({ error: MESSAGES.percent })
    .min(1, { error: MESSAGES.percent })
    .max(MAX_PERCENT, { error: MESSAGES.percent }),
  conditions,
});

const amountSchema = z.object({
  type: z.literal(BENEFIT_TYPE.AMOUNT),
  amount_ars: z
    .number({ error: MESSAGES.amount })
    .int({ error: MESSAGES.amount })
    .positive({ error: MESSAGES.amount }),
  conditions,
});

const customSchema = z.object({
  type: z.literal(BENEFIT_TYPE.CUSTOM),
  title: plain(MAX_TITLE_LENGTH, MESSAGES.titleLong).min(1, { error: MESSAGES.titleRequired }),
  conditions,
});

const benefitSchema = z.discriminatedUnion("type", [percentSchema, amountSchema, customSchema]);

/** Lo que el dueño escribió en el formulario, todo como texto. */
export interface BenefitFormValues {
  type: BenefitType;
  percent: string;
  amount: string;
  title: string;
  conditions: string;
}

export type BenefitFormResult =
  { ok: true; input: BenefitInput } | { ok: false; fields: Record<string, string> };

/** El número de un campo de texto: `NaN` si está vacío o no es un número, para que falle. */
function wholeNumber(text: string): number {
  const trimmed = text.trim();
  return /^\d+$/.test(trimmed) ? Number(trimmed) : Number.NaN;
}

/** Valida el formulario y arma lo que se envía; los errores van por nombre de campo de la API. */
export function parseBenefitForm(values: BenefitFormValues): BenefitFormResult {
  const candidate =
    values.type === BENEFIT_TYPE.PERCENT
      ? { type: values.type, percent: wholeNumber(values.percent), conditions: values.conditions }
      : values.type === BENEFIT_TYPE.AMOUNT
        ? { type: values.type, amount_ars: amountOf(values.amount), conditions: values.conditions }
        : { type: values.type, title: values.title, conditions: values.conditions };

  const parsed = benefitSchema.safeParse(candidate);
  if (!parsed.success) {
    const fields: Record<string, string> = {};
    for (const issue of parsed.error.issues) {
      const field = String(issue.path[0] ?? "type");
      fields[field] ??= issue.message;
    }
    return { ok: false, fields };
  }

  const { conditions: text, ...rest } = parsed.data;
  return { ok: true, input: text === "" ? rest : { ...rest, conditions: text } };
}

function amountOf(text: string): number {
  const amount = parsePrice(text);
  return typeof amount === "number" ? amount : Number.NaN;
}

const quotaSchema = z
  .number({ error: "Ingresá un cupo entero mayor que cero" })
  .int({ error: "Ingresá un cupo entero mayor que cero" })
  .positive({ error: "Ingresá un cupo entero mayor que cero" });

export type QuotaFormResult = { ok: true; limit: number | null } | { ok: false; message: string };

/**
 * El cupo que el dueño eligió: `null` si no quiere límite. Un cupo menor a lo ya generado
 * en el mes se rechaza acá con el número a la vista; la API lo vuelve a comprobar.
 */
export function parseQuotaForm(
  unlimited: boolean,
  text: string,
  generatedThisMonth: number,
): QuotaFormResult {
  if (unlimited) return { ok: true, limit: null };

  const parsed = quotaSchema.safeParse(wholeNumber(text));
  if (!parsed.success) return { ok: false, message: parsed.error.issues[0].message };
  if (parsed.data < generatedThisMonth) {
    return {
      ok: false,
      message: `Ya se generaron ${generatedThisMonth} cupones este mes: el cupo no puede ser menor.`,
    };
  }
  return { ok: true, limit: parsed.data };
}
