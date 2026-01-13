# Gestión de Usuarios

Esta guía explica cómo administrar los usuarios de tu firma legal en LawFlowApp.

## Vista General

El módulo de gestión de usuarios te permite controlar quién tiene acceso al sistema de tu firma, qué roles desempeñan y mantener actualizada la información de tu equipo.

## Acceso al Módulo

Para acceder a la gestión de usuarios:
1. Inicia sesión en LawFlowApp
2. Haz clic en **"Users"** en la barra de navegación superior
3. Verás una tabla con todos los usuarios de tu firma

**Nota:** Solo los usuarios con rol de **Administrador** pueden gestionar usuarios.

## Crear un Nuevo Usuario

### Pasos para Crear un Usuario

1. En la página de usuarios, haz clic en el botón **"Add User"**
2. Se abrirá un formulario con los siguientes campos:
   - **Nombre completo**: El nombre del usuario
   - **Email**: Dirección de correo electrónico (será su nombre de usuario)
   - **Contraseña**: Contraseña inicial para el usuario
   - **Rol**: Selecciona el rol apropiado (Admin, Lawyer, Staff, o Client)
   - **Estado**: Activo o Inactivo

3. Completa todos los campos requeridos
4. Haz clic en **"Create User"**
5. El usuario aparecerá inmediatamente en la tabla

### Información Importante

- **Email único**: Cada usuario debe tener un email único en el sistema
- **Contraseña temporal**: Se recomienda crear una contraseña temporal y pedirle al usuario que la cambie en su primer inicio de sesión
- **Rol por defecto**: Si no estás seguro del rol, empieza con **Staff** y actualízalo después según sea necesario

## Editar un Usuario

Para modificar la información de un usuario existente:

1. Localiza al usuario en la tabla
2. Haz clic en el botón **"Edit"** en la fila correspondiente
3. Se abrirá el formulario de edición con la información actual
4. Modifica los campos necesarios:
   - Nombre
   - Email
   - Rol
   - Estado (Activo/Inactivo)
5. Haz clic en **"Update User"** para guardar los cambios

**Nota:** No puedes cambiar la contraseña desde aquí. Los usuarios deben usar la función de "Recuperar contraseña" para cambiarla.

## Desactivar un Usuario

En lugar de eliminar usuarios, LawFlowApp permite desactivarlos. Esto preserva el historial de actividad del usuario para auditorías.

### Cuándo Desactivar un Usuario

- Cuando un empleado deja la firma
- Cuando un cliente ya no requiere acceso
- Para suspender temporalmente el acceso de alguien

### Cómo Desactivar

1. Haz clic en **"Edit"** para el usuario
2. Cambia el estado a **"Inactive"**
3. Guarda los cambios

**Efecto:** El usuario no podrá iniciar sesión, pero su información histórica permanece en el sistema.

## Tabla de Usuarios

La tabla de usuarios muestra:

| Columna | Descripción |
|---------|-------------|
| **Nombre** | Nombre completo del usuario |
| **Email** | Dirección de correo electrónico (nombre de usuario) |
| **Rol** | Rol actual asignado (Admin, Lawyer, Staff, Client) |
| **Estado** | Activo o Inactivo |
| **Acciones** | Botones para Editar o ver detalles |

### Identificación Visual de Roles

Cada rol tiene un indicador de color:
- **Admin**: Púrpura - Control total del sistema
- **Lawyer**: Azul - Acceso a casos y clientes
- **Staff**: Verde - Acceso administrativo limitado
- **Client**: Gris - Acceso restringido a sus propios casos

### Filtrado de Usuarios

Puedes filtrar la tabla por:
- **Estado**: Ver solo usuarios activos o inactivos
- **Rol**: Ver usuarios de un rol específico

