// Cómo se muestra cada error de la API del negocio. Función pura: se prueba sin React.
// Nunca se muestra el texto del error del servidor, solo estos mensajes.

export interface Feedback {
  /** Mensajes junto al campo al que se refieren, por nombre de campo de la API. */
  fields: Record<string, string>;
  /** Mensaje del formulario completo. */
  form?: FormMessage;
}

export interface FormMessage {
  title: string;
  body: string;
}

export interface Failure {
  status: number;
  error: { code: string; message: string; details?: unknown };
}

export const FIELD_MESSAGES: Record<string, string> = {
  name: "Ingresá un nombre",
  category_id: "Elegí una categoría",
  description: "La descripción tiene demasiados caracteres (máximo 500)",
  whatsapp: "Ingresá un WhatsApp válido",
  address: "Ingresá la dirección",
  city: "Ingresá la ciudad",
  location: "Marcá la ubicación en el mapa",
  booking_url: "Ingresá un link que empiece con https://",
  mp_payer_email: "Ingresá el email con el que pagaste en Mercado Pago",
  mp_payer_name: "Ingresá tu nombre completo, como figura en Mercado Pago",
  content_type: "Usá una foto JPG, PNG o WebP",
  size: "La foto debe pesar hasta 5 MB",
  price: "Ingresá un precio válido en pesos",
  reference: "La referencia puede tener hasta 100 caracteres",
  type: "Elegí un tipo de beneficio",
  percent: "Ingresá un porcentaje entero de 1 a 100",
  amount_ars: "Ingresá un monto en pesos mayor que cero",
  title: "Escribí el título del beneficio (hasta 80 caracteres)",
  conditions: "Las condiciones pueden tener hasta 120 caracteres",
  limit: "Ingresá un cupo entero mayor que cero",
};

/** Por qué se rechazó un texto libre de un beneficio (`reason` de la API). */
const TEXT_REASONS: Record<string, string> = {
  contains_link: "No incluyas links",
  contains_phone: "No incluyas teléfonos",
};

const SLOT_MESSAGES: Record<string, string> = {
  overlap: "Hay franjas que se pisan entre sí",
  invalid_time: "Revisá las horas: usá el formato HH:MM",
  invalid: "Revisá las franjas: cada una debe terminar después de empezar",
};

const GENERIC: FormMessage = {
  title: "No pudimos completar la acción",
  body: "Probá de nuevo en unos minutos.",
};

export function describeFailure({ error }: Failure): Feedback {
  switch (error.code) {
    case "validation_failed":
      return validation(error.details);

    case "business_exists":
      return form(
        "Ya tenés un negocio",
        "Cada cuenta tiene un solo negocio. Recargá la página para continuar.",
      );

    case "business_deleted":
      return form(
        "Este negocio fue dado de baja",
        "Ya no se puede modificar. Escribinos si fue un error.",
      );

    case "invalid_state":
      return form("El negocio ya fue enviado a revisión", "Te avisamos cuando esté aprobado.");

    case "plan_limit_reached":
      return form("Llegaste al límite de tu plan", limitBody(error.details));

    case "address_not_found":
      return {
        fields: { address: "No encontramos esa dirección. Revisala o ubicá el pin a mano." },
      };

    case "geocoder_busy":
      return form("La búsqueda está ocupada", "Probá de nuevo en unos segundos.");

    case "geocoder_unavailable":
      return form(
        "La búsqueda de direcciones no está disponible",
        "Ubicá el pin a mano en el mapa.",
      );

    case "image_limit_reached":
      return form("Ya subiste todas las fotos", "Borrá alguna para subir otra.");

    case "image_exists":
      return form("Esa foto ya está cargada", "Elegí otra.");

    case "upload_failed":
      return form("No pudimos subir la foto", "Probá de nuevo con otra imagen.");

    case "benefit_not_found":
      return form("Ese beneficio ya no existe", "Recargá la página para ver tus beneficios.");

    case "network_error":
      return form("Sin conexión", "Revisá tu conexión a internet y probá de nuevo.");

    case "unauthorized":
    case "invalid_session":
      return form("Tu sesión venció", "Ingresá de nuevo para continuar.");

    default:
      return { fields: {}, form: GENERIC };
  }
}

function form(title: string, body: string): Feedback {
  return { fields: {}, form: { title, body } };
}

function validation(details: unknown): Feedback {
  const { field, reason, generated_this_month } = (details ?? {}) as {
    field?: string;
    reason?: string;
    generated_this_month?: number;
  };

  if (field === "limit" && reason === "below_generated" && generated_this_month !== undefined) {
    return {
      fields: {
        limit: `Ya se generaron ${generated_this_month} cupones este mes: el cupo no puede ser menor.`,
      },
    };
  }
  const textReason = reason ? TEXT_REASONS[reason] : undefined;
  if ((field === "title" || field === "conditions") && textReason) {
    return { fields: { [field]: textReason } };
  }
  if (field === "slots") {
    return { fields: { slots: SLOT_MESSAGES[reason ?? ""] ?? SLOT_MESSAGES.invalid } };
  }
  const message = field ? FIELD_MESSAGES[field] : undefined;
  if (field && message) return { fields: { [field]: message } };

  return { fields: {}, form: GENERIC };
}

function limitBody(details: unknown): string {
  const { resource } = (details ?? {}) as { resource?: string };
  if (resource === "branches") return "Tu plan no tiene lugar para otra sede.";
  if (resource === "services") return "Tu plan no tiene lugar para otro servicio activo.";
  return "Tu plan no tiene lugar para sumar más.";
}
